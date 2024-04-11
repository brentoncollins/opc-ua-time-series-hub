package opcuaclient

import (
	"OpcUaTimeSeriesHub/hub-api/util"
	"context"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"strconv"
)

type NodeDef struct {
	NodeID      *ua.NodeID
	NodeIDParts *NodeIDParts
	NodeClass   ua.NodeClass
	BrowseName  string
	Description string
	AccessLevel ua.AccessLevelType
	Path        string
	DataType    string
	Writable    bool
	Unit        string
	Scale       string
	Min         string
	Max         string
	Children    []NodeDef
}

type NodeIDParts struct {
	Namespace      int
	IdentifierType string
	Identifier     string
}

// Records returns a slice of strings containing various properties of a NodeDef.
// The order of the properties in the resulting slice is:
// - BrowseName
// - DataType
// - String representation of the NodeID
// - Unit
// - Scale
// - Min
// - Max
// - Writable (converted to string using strconv.FormatBool)
// - Description

func (n NodeDef) Records() []string {
	return []string{n.BrowseName, n.DataType, n.NodeID.String(), n.Unit, n.Scale, n.Min, n.Max, strconv.FormatBool(n.Writable), n.Description}
}

// join concatenates two strings with a period separator.
func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}

// browse recursively retrieves the attributes and children nodes of the given OPC UA node.
func Browse(ctx context.Context, n *opcua.Node, path string, level int) ([]NodeDef, error) {

	// Set the maximum recursive tree levels to a maximum of 10
	if level > 10 {
		return nil, nil
	}

	// Get the attributes of the node
	attrs, err := n.Attributes(ctx, ua.AttributeIDNodeClass, ua.AttributeIDBrowseName, ua.AttributeIDDescription, ua.AttributeIDAccessLevel, ua.AttributeIDDataType)
	if err != nil {
		util.Logger.Errorf("Failed the get the attributes of node: %s", n.ID.String())
		return nil, err
	}

	// Set the definition of the node starting with the NodeID
	var def = NodeDef{
		NodeID: n.ID,
	}

	// Get the Node Class
	switch err := attrs[0].Status; err {
	case ua.StatusOK:
		def.NodeClass = ua.NodeClass(attrs[0].Value.Int())
	default:
		return nil, err
	}

	// Get the node browse name
	switch err := attrs[1].Status; err {
	case ua.StatusOK:
		def.BrowseName = attrs[1].Value.String()
	default:
		return nil, err
	}

	// Get the node description
	switch err := attrs[2].Status; err {
	case ua.StatusOK:
		def.Description = attrs[2].Value.String()
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	// Get the node Access Level
	switch err := attrs[3].Status; err {
	case ua.StatusOK:
		def.AccessLevel = ua.AccessLevelType(attrs[3].Value.Int())
		def.Writable = def.AccessLevel&ua.AccessLevelTypeCurrentWrite == ua.AccessLevelTypeCurrentWrite
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	// Get the node Data Type
	switch err := attrs[4].Status; err {
	case ua.StatusOK:
		switch v := attrs[4].Value.NodeID().IntID(); v {
		case id.DateTime:
			def.DataType = "time.Time"
		case id.Boolean:
			def.DataType = "bool"
		case id.SByte:
			def.DataType = "int8"
		case id.Int16:
			def.DataType = "int16"
		case id.Int32:
			def.DataType = "int32"
		case id.Byte:
			def.DataType = "byte"
		case id.UInt16:
			def.DataType = "uint16"
		case id.UInt32:
			def.DataType = "uint32"
		case id.UtcTime:
			def.DataType = "time.Time"
		case id.String:
			def.DataType = "string"
		case id.Float:
			def.DataType = "float32"
		case id.Double:
			def.DataType = "float64"
		default:
			def.DataType = attrs[4].Value.NodeID().String()
		}
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	// Set the node Path
	def.Path = join(path, def.BrowseName)
	// fmt.Printf("%d: def.Path:%s def.NodeClass:%s\n", level, def.Path, def.NodeClass)

	var nodes []NodeDef
	if def.NodeClass == ua.NodeClassVariable || def.NodeClass == ua.NodeClassObject {
		nodes = append(nodes, def)
	}

	// Browse children
	browseChildren := func(refType uint32) error {
		refs, err := n.ReferencedNodes(ctx, refType, ua.BrowseDirectionForward, ua.NodeClassAll, true)
		if err != nil {
			return errors.Errorf("References: %d: %s", refType, err)
		}
		for _, rn := range refs {
			children, err := Browse(ctx, rn, join(path, def.BrowseName), level+1)
			if err != nil {
				return errors.Errorf("browse children: %s", err)
			}
			def.Children = append(def.Children, children...) // Add children directly to the parent node
		}
		return nil
	}

	if err := browseChildren(id.HasComponent); err != nil {
		return nil, err
	}
	if err := browseChildren(id.Organizes); err != nil {
		return nil, err
	}
	if err := browseChildren(id.HasProperty); err != nil {
		return nil, err
	}

	return []NodeDef{def}, nil
}
