package webapi

import (
	"OpcUaTimeSeriesHub/hub-api/internal/configupdate"
	"OpcUaTimeSeriesHub/hub-api/internal/database"
	"OpcUaTimeSeriesHub/hub-api/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var RequestData struct {
	NodeID         string `json:"nodeID"`
	HistoryEnabled bool   `json:"historyEnabled"`
	NodePath       string `json:"nodePath"`
}

// GetNodesHandler handles requests for the nodes hierarchy.
func GetNodesHandler(w http.ResponseWriter, r *http.Request) {

	util.Logger.Info("Getting nodes hierarchy")

	db, err := database.LoadDatabase()
	if err != nil {
		util.Logger.Error("Failed to initialize DB", err)
		http.Error(w, "Failed to initialize DB", http.StatusInternalServerError)
		return
	}

	roots, err := database.LoadHierarchy(db)
	if err != nil {
		util.Logger.Error("Failed to load node hierarchy", err)
		http.Error(w, "Failed to load node hierarchy", http.StatusInternalServerError)
		return
	}

	telegrafUpToDate, err := database.IsTelegrafUpToDate(db)
	if err != nil {
		util.Logger.Error("Failed to telegraf current state from database", err)
		http.Error(w, "Failed to telegraf current state from database", http.StatusInternalServerError)
		return
	}

	util.Logger.Info("Successfully loaded node hierarchy")
	util.Logger.Info("Successfully loaded telegraf current state from database")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Nodes            []*database.Node `json:"nodes"`
		TelegrafUpToDate bool             `json:"telegrafUpToDate"`
	}{roots, telegrafUpToDate})

}

func UpdateNodeHistoryHandler(w http.ResponseWriter, r *http.Request) {

	util.Logger.Info("Updating node history")

	db, err := database.LoadDatabase()
	if err != nil {
		util.Logger.Error("Failed to initialize DB", err)
		http.Error(w, "Failed to initialize DB", http.StatusInternalServerError)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&RequestData); err != nil {
		util.Logger.Error("Invalid request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = database.UpdateNodeHistory(db, RequestData.NodeID, RequestData.HistoryEnabled)
	if err != nil {
		util.Logger.Error("Failed to update node", err)
		http.Error(w, "Failed to update node", http.StatusInternalServerError)
		return
	}

	var modifyType string

	if RequestData.HistoryEnabled {
		modifyType = "Enabled"
	} else {
		modifyType = "Disabled"
	}

	util.Logger.WithField("NodeID", RequestData.NodeID).WithField("HistoryEnabled", RequestData.HistoryEnabled).Info("Node history updated")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(
		struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}{"success", fmt.Sprintf("History for node %s: %s", RequestData.NodePath, modifyType)})
}

func UpdateConfigFileWithHistoryNodes(w http.ResponseWriter, r *http.Request) {

	util.Logger.Info("Updating config file with history nodes")

	db, err := database.LoadDatabase()
	if err != nil {
		util.Logger.Error("Failed to initialize DB", err)
		http.Error(w, "Failed to initialize DB", http.StatusInternalServerError)
		return
	}

	nodes, err := database.GetHistoryNodes(db)
	if err != nil {
		util.Logger.Error("Failed to fetch nodes", err)
		http.Error(w, "Failed to fetch nodes", http.StatusInternalServerError)
	}
	telegrafConfigFilePath := os.Getenv("TELEGRAF_CONFIG_FILE")

	// This will update the configuration with all existing and added nodes
	err = configupdate.UpdateConfig(telegrafConfigFilePath, database.FilterNodesByAction(nodes, []database.Action{database.Added, database.HistoryEnabledNoChange}))
	if err != nil {
		util.Logger.Error("Failed to update config file", err)
		http.Error(w, "Failed to update config file", http.StatusInternalServerError)
		return
	}

	err = database.SetDatabaseNodeStates(db, nodes)
	if err != nil {
		util.Logger.Error("Failed to update node states in database", err)
		http.Error(w, "Failed to update node states in database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(
		struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}{"success", fmt.Sprintf("Config file updated with history nodes")})

}

func GetUpdatesRequired(w http.ResponseWriter, r *http.Request) {

	util.Logger.Info("Getting updates required")

	db, err := database.LoadDatabase()
	if err != nil {
		util.Logger.Error("Failed to initialize DB", err)
		http.Error(w, "Failed to initialize DB", http.StatusInternalServerError)
		return
	}
	nodes, err := database.GetHistoryNodes(db)
	if err != nil {
		util.Logger.Error("Failed to fetch nodes", err)
		http.Error(w, "Failed to fetch nodes", http.StatusInternalServerError)
	}

	modifiedNodes := database.FilterNodesByAction(nodes, []database.Action{database.Added, database.Removed})
	if err != nil {
		util.Logger.Error("Failed to fetch nodes", err)
		http.Error(w, "Failed to fetch nodes", http.StatusInternalServerError)
	}

	// Create a new slice of anonymous structs containing only NodeID and DBActionRequired
	output := make([]struct {
		NodeID           string `json:"nodeID"`
		DBActionRequired string `json:"dbActionRequired"`
	}, len(modifiedNodes))

	for i, node := range modifiedNodes {
		output[i].NodeID = node.BrowseName
		output[i].DBActionRequired = node.DBActionRequired.String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

// GetSingleNodeHandler handles requests for a single node's data
//func GetSingleNodeHandler(w http.ResponseWriter, r *http.Request) {
//
//	util.Logger.Info("Getting single node")
//
//	db, err := database.LoadDatabase()
//	if err != nil {
//		util.Logger.Error("Failed to initialize DB", err)
//		http.Error(w, "Failed to initialize DB", http.StatusInternalServerError)
//		return
//	}
//
//	vars := mux.Vars(r)
//	nodeID := vars["nodeID"]
//
//	// Function to query your database for a single node by NodeID
//	node, err := database.GetSingleNode(db, nodeID)
//	if err != nil {
//		http.Error(w, "Failed to find node", http.StatusNotFound)
//		return
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(node)
//}
