package database

import (
	"OpcUaTimeSeriesHub/hub-api/internal/opcuaclient"
	"OpcUaTimeSeriesHub/hub-api/util"
	"context"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
	"log"
	"regexp"
	"time"
)

type Action int

func LoadDatabase() (*sql.DB, error) {

	config := util.LoadConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.DbUser,
		config.DbPassword, config.DbHost, config.DbPort, config.DbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		util.Logger.Error("Error opening database", err)
		return nil, err
	}
	util.Logger.Info("Database opened successfully")
	return db, nil
}

// InitDB initializes and returns the database connection.
func InitDB(dropTable bool) (*sql.DB, error) {

	util.Logger.Info("Initializing database")

	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = LoadDatabase()
		if err == nil {
			break
		}
		util.Logger.Errorf("Attempt %d: Failed to connect: %s", i+1, err)
		time.Sleep(time.Second * 2)
	}

	if err != nil {
		util.Logger.Error("Failed to connect to database after 10 attempts", err)
		// handle error
	}

	if err != nil {
		util.Logger.Error("Error opening database", err)
		return nil, err
	}

	if dropTable {
		err = DropNodesTable(db)
		if err != nil {
			util.Logger.Error("Error dropping nodes table", err)
			return nil, err
		}
	}

	createNodesSQL := `
CREATE TABLE IF NOT EXISTS nodes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    node_id TEXT(500) NOT NULL,
    namespace TEXT,
    identifier_type VARCHAR(255),
    identifier TEXT,
    parent_id TEXT,
    browse_name TEXT,
    node_class VARCHAR(255),
    data_type VARCHAR(255),
    writable INT,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    removed INT DEFAULT 0,
    node_path TEXT,
    history_enabled INT DEFAULT 0,
    included_in_config INT DEFAULT 0,
    UNIQUE(node_id(500))
);
`

	// Execute the SQL statement to create the table
	_, err = db.Exec(createNodesSQL)
	if err != nil {
		util.Logger.Error("Error creating nodes table", err)
		return nil, err
	}

	util.Logger.Info("Tables created successfully")
	return db, nil
}

func InsertOrUpdateNode(db *sql.DB, node Node) error {
	// Prepare the SQL statement
	statement := `
    INSERT INTO nodes (node_id, namespace, identifier_type, identifier, parent_id, browse_name, node_class, data_type, writable, node_path, history_enabled, included_in_config, removed)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON DUPLICATE KEY UPDATE
    parent_id = VALUES(parent_id),
    browse_name = VALUES(browse_name),
    node_class = VALUES(node_class),
    data_type = VALUES(data_type),
    writable = VALUES(writable),
    node_path = VALUES(node_path),
    history_enabled = VALUES(history_enabled),
    included_in_config = VALUES(included_in_config),
	removed = VALUES(removed);
    
`

	// Execute the SQL statement with the provided parameters
	_, err := db.Exec(statement, node.NodeID, node.Namespace, node.IdentifierType, node.Identifier, node.ParentID, node.BrowseName, node.NodeClass, node.DataType, node.Writable, node.NodePath, node.HistoryEnabled, node.HistoryEnabledInConfig, node.Removed)
	if err != nil {
		// If there's an error, wrap it with additional context and return
		return fmt.Errorf("inserting or updating node: %w", err)
	}

	return nil
}

func MarkNodeAsRemoved(db *sql.DB, nodeID string) error {

	util.Logger.Info("Marking node as removed", nodeID)
	_, err := db.Exec("UPDATE nodes SET removed = 1, last_updated = CURRENT_TIMESTAMP WHERE node_id = ?", nodeID)
	if err != nil {
		util.Logger.Error("Error marking node as removed", err)
	}
	_, err = db.Exec("UPDATE nodes SET history_enabled = 0, last_updated = CURRENT_TIMESTAMP WHERE node_id = ?", nodeID)
	if err != nil {
		util.Logger.Error("Error disabling history on node", err)
	}

	return err
}

// DropNodesTable drops the 'nodes' table from the database.
// Use with caution - this will remove all data in the 'nodes' table.
func DropNodesTable(db *sql.DB) error {
	dropTableSQL := `DROP TABLE IF EXISTS nodes;`
	_, err := db.Exec(dropTableSQL)
	if err != nil {
		log.Printf("Error dropping nodes table: %s", err)
		return err
	}
	log.Println("nodes table dropped successfully")
	return nil
}

// LoadHierarchy loads the node hierarchy from the database, adjusting for string ParentID.
func LoadHierarchy(db *sql.DB) ([]*Node, error) {
	// Temporary map to hold nodes by NodeID
	nodesMap := make(map[string]*Node)
	// Slice to hold all root nodes (there could be multiple roots)
	var roots []*Node

	// Query all nodes from the database
	rows, err := db.Query(`SELECT id, node_id, parent_id, browse_name, node_class, data_type, writable, last_updated, removed, node_path, history_enabled FROM nodes WHERE removed = 0 ORDER BY browse_name`)
	if err != nil {
		return nil, fmt.Errorf("querying nodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Node
		err := rows.Scan(&n.ID, &n.NodeID, &n.ParentID, &n.BrowseName, &n.NodeClass, &n.DataType, &n.Writable, &n.LastUpdated, &n.Removed, &n.NodePath, &n.HistoryEnabled)
		if err != nil {
			return nil, fmt.Errorf("scanning node: %w", err)
		}

		nodesMap[n.NodeID] = &n

		// If ParentNodeID is its own NodeID, it's a root node
		if n.NodeID == n.ParentID {
			roots = append(roots, &n)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading rows: %w", err)
	}

	// Build the hierarchy
	for _, n := range nodesMap {
		if n.ParentID != n.NodeID { // Ignore root nodes
			parent, exists := nodesMap[n.ParentID]
			if !exists {
				// Handle cases where a parent node might not exist in the map
				continue
			}
			parent.Children = append(parent.Children, n)
		}
	}

	// Return the roots of the hierarchy, which now includes child nodes
	return roots, nil
}

// UpdateNodeHistory updates the history_enabled status of a node identified by nodeID.
func UpdateNodeHistory(db *sql.DB, nodeID string, historyEnabled bool) error {
	// Convert the boolean to an integer because SQLite does not have a native boolean type
	historyEnabledInt := 0
	if historyEnabled {
		historyEnabledInt = 1
	}

	statement := `UPDATE nodes SET history_enabled = ? WHERE node_id = ?`
	_, err := db.Exec(statement, historyEnabledInt, nodeID)
	if err != nil {
		// If there's an error, wrap it with additional context and return
		return fmt.Errorf("updating node history_enabled: %w", err)
	}

	return nil
}

// GetHistoryNodes retrieves all nodes from the database where history is enabled or config is enabled
func GetHistoryNodes(db *sql.DB) ([]*Node, error) {
	query := `SELECT 
					n.id,
					n.node_id,
					n.namespace,
					n.identifier_type,
					n.identifier,
					n.parent_id,
					n.browse_name,
					n.node_class,
					n.data_type,
					n.writable,
					n.node_path,
					n.history_enabled,
					n.included_in_config,
					CASE 
						WHEN history_enabled = 1 AND included_in_config = 0 THEN 'Added'
						WHEN history_enabled = 0 AND included_in_config = 1 THEN 'Removed'
					    WHEN history_enabled = 1 AND included_in_config = 1 THEN 'HistoryEnabledNoChange'
					    WHEN history_enabled = 0 AND included_in_config = 0 THEN 'HistoryDisabledNoChange'
					END AS status
				FROM 
					nodes n`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying history-enabled nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		var node Node
		var status sql.NullString
		if err := rows.Scan(&node.ID, &node.NodeID, &node.Namespace, &node.IdentifierType, &node.Identifier, &node.ParentID, &node.BrowseName, &node.NodeClass, &node.DataType, &node.Writable, &node.NodePath, &node.HistoryEnabled, &node.HistoryEnabledInConfig, &status); err != nil {
			return nil, fmt.Errorf("scanning node: %w", err)
		}
		switch status.String {
		case "Added":
			node.DBActionRequired = Added
		case "Removed":
			node.DBActionRequired = Removed
		case "HistoryEnabledNoChange":
			node.DBActionRequired = HistoryEnabledNoChange
		case "HistoryDisabledNoChange":
			node.DBActionRequired = HistoryDisabledNoChange
		default:
			node.DBActionRequired = NoAction
		}
		nodes = append(nodes, &node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return nodes, nil
}

// IsTelegrafUpToDate checks the current update state in the telegraf_config_state table.
func IsTelegrafUpToDate(db *sql.DB) (bool, error) {
	util.Logger.Info("Checking if Telegraf config is up to date")

	// Query to check if there are any nodes that need to be added or removed
	query := `SELECT 
		n.*, 
		CASE 
			WHEN history_enabled = 1 AND included_in_config = 0 THEN 'Added'
			WHEN history_enabled = 0 AND included_in_config = 1 THEN 'Removed'
		END AS status
	FROM 
		nodes n 
	WHERE 
		(history_enabled = 1 AND included_in_config = 0)
		OR (history_enabled = 0 AND included_in_config = 1)`

	rows, err := db.Query(query)
	if err != nil {
		return false, fmt.Errorf("querying nodes: %w", err)
	}
	defer rows.Close()

	// If the query returns any rows, then Telegraf is not up-to-date
	if rows.Next() {
		util.Logger.Info("Telegraf config is not up to date")
		return false, nil
	}

	util.Logger.Info("Telegraf config is up to date")
	return true, nil
}

// SetDatabaseNodeStates updates the current update state in the telegraf_config_state table to 1.
func SetDatabaseNodeStates(db *sql.DB, nodes []*Node) error {

	for _, node := range nodes {
		util.Logger.Info("Updating node in database", node.NodeID, "with status", node.DBActionRequired)
		if node.DBActionRequired == Added {
			_, err := db.Exec("UPDATE nodes SET included_in_config = 1 WHERE node_id = ?", node.NodeID)
			if err != nil {
				util.Logger.Error("Error setting Telegraf config to up to date", err)
				return fmt.Errorf("updating update state: %w", err)
			}

		}
		if node.DBActionRequired == Removed {
			_, err := db.Exec("UPDATE nodes SET included_in_config = 0 WHERE node_id = ?", node.NodeID)
			if err != nil {
				util.Logger.Error("Error setting Telegraf config to up to date", err)
				return fmt.Errorf("updating update state: %w", err)
			}
		}
	}

	util.Logger.Info("Telegraf config set to up to date successfully")
	return nil
}

func UpdateHierarchy() error {

	config := util.LoadConfig()

	db, err := LoadDatabase()
	if err != nil {
		util.Logger.Error("Failed to initialize DB", err)
		return err
	}

	endpoint := flag.String("endpoint", config.TelegrafOpcUaEndpoint, "OPC UA Endpoint URL")
	nodeID := flag.String("node", config.RootNode, "node id for the root node")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	c, err := opcua.NewClient(*endpoint)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		err = c.Connect(ctx)
		if err == nil {
			break
		}
		util.Logger.Errorf("Attempt %d: Failed to connect: %s", i+1, err)
		time.Sleep(time.Second * 2)
	}

	if err != nil {
		log.Fatalf("Failed to connect after 10 attempts: %s", err)
	}

	defer c.Close(ctx)

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		log.Fatalf("invalid node id: %s", err)
	}

	nodeList, err := opcuaclient.Browse(ctx, c.Node(id), "", 0)
	if err != nil {
		util.Logger.Errorf("Failed to browse: %s", err)
		log.Fatal(err)
	}

	util.Logger.Info("Checking for removed nodes form the OPC UA Server")

	_, err = insertNodesRecursively(db, nodeList, config.RootNode)
	if err != nil {
		util.Logger.Error("Failed to insert nodes", err)
		return err
	}

	// Need to work on this

	//err = MarkRemovedNodes(db, allOpcNodes)
	//if err != nil {
	//	util.Logger.Error("Failed to mark removed nodes", err)
	//	return err
	//}

	return err

}

// ParseNodeIDString parses the string representation of a NodeID.
func ParseNodeIDString(nodeIDStr string) (*opcuaclient.NodeIDParts, error) {
	// Regular expression to match the NodeID string format
	// It has three capture groups: namespace (ns), namespace identifier (s, g, b, i), and identifier
	re := regexp.MustCompile(`^(?:ns=(\d+);)?([isgb]=)?(.+)$`)

	matches := re.FindStringSubmatch(nodeIDStr)
	if matches == nil {
		return nil, fmt.Errorf("invalid NodeID format")
	}

	// Create a NodeIDParts instance to hold the parsed parts
	parts := &opcuaclient.NodeIDParts{}

	// Populate the Namespace if present
	if matches[1] != "" {
		fmt.Sscanf(matches[1], "%d", &parts.Namespace)
	}

	// Remove '=' from the namespace identifier type if present
	if len(matches[2]) > 0 {
		parts.IdentifierType = matches[2][:1] // Take only the first character (s, g, b, or i)
	}

	parts.Identifier = matches[3]

	return parts, nil
}

func insertNodesRecursively(db *sql.DB, nodes []opcuaclient.NodeDef, parentID string) ([]Node, error) {
	var result []Node

	for _, node := range nodes {
		util.Logger.Debugf("Inserting node: %s", node.NodeID)
		NodeID := node.NodeID.String()
		node.NodeIDParts, _ = ParseNodeIDString(NodeID)

		// Insert the current node, using the provided parentID
		err := InsertOrUpdateNode(db, Node{
			NodeID:                 node.NodeID.String(),
			Namespace:              node.NodeIDParts.Namespace,
			IdentifierType:         node.NodeIDParts.IdentifierType,
			Identifier:             node.NodeIDParts.Identifier,
			ParentID:               parentID,
			BrowseName:             node.BrowseName,
			NodeClass:              node.NodeClass.String(),
			DataType:               node.DataType,
			Writable:               node.Writable,
			NodePath:               node.Path,
			Removed:                false,
			HistoryEnabled:         false,
			HistoryEnabledInConfig: false,
			DBActionRequired:       NoAction,
		})
		if err != nil {
			return nil, err
		}

		// Add the current node to the result slice
		result = append(result, Node{
			NodeID:                 node.NodeID.String(),
			Namespace:              node.NodeIDParts.Namespace,
			IdentifierType:         node.NodeIDParts.IdentifierType,
			Identifier:             node.NodeIDParts.Identifier,
			ParentID:               parentID,
			BrowseName:             node.BrowseName,
			NodeClass:              node.NodeClass.String(),
			DataType:               node.DataType,
			Writable:               node.Writable,
			NodePath:               node.Path,
			HistoryEnabled:         false,
			HistoryEnabledInConfig: false,
			DBActionRequired:       NoAction,
		})

		// Recursively insert the children of the current node
		children, err := insertNodesRecursively(db, node.Children, node.NodeID.String())
		if err != nil {
			return nil, err
		}

		// Add the children to the result slice
		result = append(result, children...)
	}

	return result, nil
}

//func MarkRemovedNodes(db *sql.DB, opcuaNodes []Node) error {
//	// Retrieve all nodes from the database
//	dbNodes, err := LoadHierarchy(db)
//	if err != nil {
//		return fmt.Errorf("loading hierarchy from database: %w", err)
//	}
//
//	// Convert OPC UA nodes to a map for efficient lookup
//	opcuaNodeMap := make(map[string]struct{})
//	for _, node := range opcuaNodes {
//		opcuaNodeMap[node.NodeID] = struct{}{}
//	}
//
//	util.Logger.Debugf("OPC UA Nodes: %v", opcuaNodeMap)
//
//	// Call the recursive function to mark nodes as removed
//	err = markNodesRecursively(db, dbNodes, opcuaNodeMap)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

//func markNodesRecursively(db *sql.DB, dbNodes []*Node, opcuaNodeMap map[string]struct{}) error {
//	// Iterate over the database nodes
//	for _, dbNode := range dbNodes {
//		util.Logger.Debugf("Checking node: %s", dbNode.NodeID)
//		// If a database node doesn't exist in the OPC UA nodes, mark it as removed
//		if _, exists := opcuaNodeMap[dbNode.NodeID]; !exists {
//			err := MarkNodeAsRemoved(db, dbNode.NodeID)
//			if err != nil {
//				return fmt.Errorf("marking node as removed: %w", err)
//			}
//		}
//
//		// Recursively check the children of the current node
//		err := markNodesRecursively(db, dbNode.Children, opcuaNodeMap)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
