package database

const (
	NoAction Action = iota
	Added
	Removed
	HistoryEnabledNoChange
	HistoryDisabledNoChange
)

func (a Action) String() string {
	switch a {
	case NoAction:
		return "No Action"
	case Added:
		return "Added"
	case Removed:
		return "Removed"
	case HistoryEnabledNoChange:
		return "History Enabled No Change"
	case HistoryDisabledNoChange:
		return "History Disabled No Change"
	default:
		return "Unknown"
	}
}

// Node represents a node in the OPC UA hierarchy.
type Node struct {
	ID                     int
	NodeID                 string
	ParentID               string
	BrowseName             string
	NodeClass              string
	DataType               string
	Writable               bool
	HistoryEnabled         bool
	LastUpdated            string
	Removed                bool
	Children               []*Node
	Namespace              int
	IdentifierType         string
	Identifier             string
	NodePath               string
	HistoryEnabledInConfig bool
	DBActionRequired       Action
}

type SimpleNode struct {
	NodeID string
}

func FilterNodesByAction(nodes []*Node, actions []Action) []*Node {
	var filteredNodes []*Node

	for _, node := range nodes {
		for _, action := range actions {
			if node.DBActionRequired == action {
				filteredNodes = append(filteredNodes, node)
				break
			}
		}
	}

	return filteredNodes
}
