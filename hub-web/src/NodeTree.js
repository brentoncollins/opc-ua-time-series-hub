// Import necessary modules from React, Axios for HTTP requests, and Font Awesome for icons
import React, {useState, useEffect, useCallback} from 'react';
import axios from 'axios';
import './App.css'; // Importing CSS for styling
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'; // Component for icons
import { faPlusSquare, faMinusSquare } from '@fortawesome/free-solid-svg-icons'; // Icons for expand/collapse
import Modal from 'react-modal'; // Importing Modal from react-modal

Modal.setAppElement('#root');
// Main functional component for rendering the node tree
const NodeTree = () => {
    // State for storing all nodes, expanded nodes, and the current search term
    const [nodes, setNodes] = useState([]);
    const [expandedNodes, setExpandedNodes] = useState(new Set());
    const [searchTerm, setSearchTerm] = useState('');
    const [telegrafUpdateRequired, setTelegrafUpdateRequired] = useState('');
    const [responseMessage, setResponseMessage] = useState('');
    const [isErrorMessage, setIsErrorMessage] = useState(false);
    const [modalIsOpen, setModalIsOpen] = useState(false); // State for controlling the modal
    const [updatesRequired, setUpdatesRequired] = useState([]); // State for storing the updates required
    const [errorMessage, setErrorMessage] = useState(null);

    // Function to initially set or update node visibility
    const augmentNodesWithVisibility = useCallback((nodes, isVisible) => {
        return nodes.map(node => ({
            ...node,
            visible: isVisible,
            Children: node.Children ? augmentNodesWithVisibility(node.Children, isVisible) : [],
        }));
    }, []);

    // Function to dynamically update node visibility based on a search term
    const updateNodeVisibilityBasedOnSearch = useCallback((nodes, term) => {
        const termLower = term.toLowerCase();
        return nodes.map(node => {
            // Checks if node's path includes the search term and updates visibility accordingly
            const matchesSearch = node.NodePath.toLowerCase().includes(termLower);
            const childNodes = node.Children ? updateNodeVisibilityBasedOnSearch(node.Children, term) : [];
            const hasVisibleChildren = childNodes.some(child => child.visible);
            return {
                ...node,
                visible: matchesSearch || hasVisibleChildren,
                Children: childNodes,
            };
        });
    }, []);

    // Effect hook to fetch nodes on component mount
    useEffect(() => {
        axios.get('api/nodes')
            .then(response => {
                const nodes = response.data.nodes;
                const telegrafUpToDate = response.data.telegrafUpToDate;
                // Augment fetched nodes with visibility property
                const augmentedNodes = augmentNodesWithVisibility(nodes, true);
                setTelegrafConfigStatus(telegrafUpToDate);
                setNodes(augmentedNodes);
                setErrorMessage(null); // Reset error message on successful fetch
            })
            .catch(error => {
                console.error('Error fetching nodes: OPC UA Server likely starting refresh in a couple of minutes.', error);
                // Step 2: Update the error message state
                setErrorMessage('Failed to fetch nodes. Please try again later.');
            });
    }, [augmentNodesWithVisibility]);

    // Effect hook to filter nodes based on the search term
    useEffect(() => {
        if (searchTerm.trim() === '') {
            // If search term is empty, make all nodes visible
            setNodes(prevNodes => augmentNodesWithVisibility(prevNodes, true));
        } else {
            // Update node visibility based on the search term
            setNodes(prevNodes => updateNodeVisibilityBasedOnSearch(prevNodes, searchTerm));
        }
    }, [searchTerm, augmentNodesWithVisibility, updateNodeVisibilityBasedOnSearch]);


    // Function to send a request to update Telegraf config (not fully detailed in the provided code snippet)
    const updateTelegrafConfig = () => {
        // Example HTTP POST request using Axios
        axios.post('api/update-telegraf-config', { /* payload if needed */ })
          .then(response => {
            console.log('Telegraf config updated:', response.data);
            setTelegrafConfigStatus(true);
            setModalIsOpen(false);
          })
          .catch(error => {
            setModalIsOpen(false);
            console.error('Failed to update Telegraf config:', error);
          });
      };

    const getUpdatesRequired = () => {
        // Example HTTP GET request using Axios
        axios.get('api/updated-required')
            .then(response => {
                console.log('Updates required:', response.data);
                setUpdatesRequired(response.data);
                setModalIsOpen(true);
            })
            .catch(error => {
                console.error('Failed to get updates required:', error);
            });
    };


    // Function to find a node by ID and update its history enabled state
    const findAndUpdateNode = (nodes, nodeID, isChecked) => {
        return nodes.map(node => {
          if (node.NodeID === nodeID) {
            // Logs the update and returns the node with updated history enabled state
            return { ...node, HistoryEnabled: isChecked };
          } else if (node.Children) {
            // Recursively updates children nodes
            return { ...node, Children: findAndUpdateNode(node.Children, nodeID, isChecked) };
          }
          return node;
        });
      };

      const setTelegrafConfigStatus = (telegrafUpToDate) => {
        if (telegrafUpToDate) {
            setTelegrafUpdateRequired('Telegraf config is up to date.');
          setIsErrorMessage(false);
        } else {
            setTelegrafUpdateRequired('Telegraf config requires an update.');
            setIsErrorMessage(true);
            }
      }

      const toggleHistoryEnabled = (nodeID, isChecked, nodePath) => {
        axios.post('/api/update-node-history', { nodeID, historyEnabled: isChecked, nodePath })
          .then(response => {
            // Assuming the response includes a status and message
            if (response.data && response.data.status === 'success') {
            // Updates the node state locally without re-fetching from the server
            setNodes(currentNodes => findAndUpdateNode(currentNodes, nodeID, isChecked));
            setResponseMessage(response.data.message); // Set the success message
            setIsErrorMessage(false); // Indicate that this is not an error message
            setTelegrafConfigStatus(false); // If the update was a success, we konw it is not up to date. No need to read the database. 
            } else {
              // Handle any case where the status is not success as an error
              setResponseMessage('An unexpected error occurred.');
              setIsErrorMessage(true);
            }
            // Optionally, refresh nodes here if needed
          })
          .catch(error => {
            console.error('Error updating node:', error);
            setResponseMessage(error.message || 'Failed to update node.'); // Set the error message
            setIsErrorMessage(true); // Indicate that this is an error message
          });
      };

    // Function to toggle the expansion of nodes to show/hide children
    const toggleExpansion = (nodeID) => {
        // Updates the set of expanded nodes based on user interaction
        setExpandedNodes(expandedNodes => {
            const newExpandedNodes = new Set(expandedNodes);
            if (newExpandedNodes.has(nodeID)) {
                newExpandedNodes.delete(nodeID);
            } else {
                newExpandedNodes.add(nodeID);
            }
            return newExpandedNodes;
        });
    };

const renderNode = (node, depth = 0) => {
    if (!node.visible) return null;

    const hasChildren = node.Children && node.Children.length > 0;
    const isExpanded = expandedNodes.has(node.NodeID);
    const tooltipText = `Path: ${node.NodePath}\nID: ${node.NodeID}`;

    // Calculate indentation based on depth
    const marginLeft = depth * 10; // 10px per depth level

    return (
        <div key={node.NodeID} title={tooltipText} style={{ marginLeft: `${marginLeft}px` }}>
            <div className="node-item">
                <div className="node-content">
                    {hasChildren && (
                        <span className="toggle-btn" onClick={() => toggleExpansion(node.NodeID)}>
                            {isExpanded ? <FontAwesomeIcon icon={faMinusSquare} /> : <FontAwesomeIcon icon={faPlusSquare} />}
                        </span>
                    )}
                    <span>{node.BrowseName}</span>
                </div>
                <div className="node-details">
                    {node.DataType && <span className="node-data-type">{node.DataType}</span>}
                    {node.NodeClass === "NodeClassVariable" && (
                        <input
                            type="checkbox"
                            className="history-checkbox"
                            checked={node.HistoryEnabled}
                            onChange={(e) => toggleHistoryEnabled(node.NodeID, e.target.checked, node.NodePath)}
                        />
                    )}
                </div>
            </div>
            {isExpanded && hasChildren && (
                <div>{node.Children.map((child, index) => renderNode(child, depth + 1))}</div>
            )}
        </div>
    );
};

    return (
        <div className="node-tree-container">
            <div className="controls-container">
                <input
                    type="text"
                    placeholder="Search nodes..."
                    value={searchTerm}
                    onChange={e => setSearchTerm(e.target.value)}
                    className="search-box"
                />
                <button onClick={getUpdatesRequired} className="update-config-button">
                    Update Telegraf Config
                </button>
            </div>
            <div className="response-message-container">
                {responseMessage ? (
                    <div className={isErrorMessage ? 'error-message' : 'success-message'}>
                        {responseMessage}
                    </div>
                ) : (
                    <div className="spacer"></div>
                )}
                <div className='update-telegraf-message'>
                    {telegrafUpdateRequired}
                </div>
            </div>
            <div className="node-tree">
                {errorMessage && <div className="error-message">{errorMessage}</div>}
                {nodes.map(node => renderNode(node))}
            </div>
            <Modal
                isOpen={modalIsOpen}
                onRequestClose={() => setModalIsOpen(false)}
                contentLabel="Updates Required"
                style={{ content: { width: '50%', height: '75%', margin: 'auto' }}}
            >
                <h2>Updates Required</h2>
                <div className="modal-buttons">
                    <button onClick={updateTelegrafConfig} className="modal-button ok-button">OK</button>
                    <button onClick={() => setModalIsOpen(false)} className="modal-button cancel-button">Cancel</button>
                </div>
                <div className="scrollable-table-container">
                    <table className="updates-required">
                        <thead>
                        <tr>
                            <th>Node ID</th>
                            <th>Action Required</th>
                        </tr>
                        </thead>
                        <tbody>
                        {updatesRequired.map((update, index) => (
                            <tr key={index}>
                                <td>{update.nodeID}</td>
                                <td>{update.dbActionRequired}</td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            </Modal>
        </div>
    );
}


export default NodeTree;
