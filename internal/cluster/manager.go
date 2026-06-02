package cluster

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type NodeManager struct {
	mu      sync.RWMutex
	nodes   map[string]*Node
	clients map[string]AgentClient
	stopCh  chan struct{}
}

func NewNodeManager() *NodeManager {
	return &NodeManager{
		nodes:   make(map[string]*Node),
		clients: make(map[string]AgentClient),
		stopCh:  make(chan struct{}),
	}
}

func (nm *NodeManager) AddLocalNode(client AgentClient) (*Node, error) {
	hostname, _ := os.Hostname()
	return nm.addNode("local", hostname, client)
}

func (nm *NodeManager) AddRemoteNode(id, name string, client AgentClient) (*Node, error) {
	return nm.addNode(id, name, client)
}

func (nm *NodeManager) addNode(id, name string, client AgentClient) (*Node, error) {
	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("ping %s: %w", name, err)
	}

	nm.mu.Lock()
	defer nm.mu.Unlock()

	if existing, ok := nm.nodes[id]; ok {
		existing.Status = NodeConnected
		nm.clients[id] = client
		fmt.Printf("Node reconnected: %s (%s)\n", name, id)
		return existing, nil
	}

	node := &Node{
		ID:     id,
		Name:   name,
		Status: NodeConnected,
	}
	nm.nodes[id] = node
	nm.clients[id] = client
	fmt.Printf("Node added: %s (%s)\n", name, id)
	return node, nil
}

func (nm *NodeManager) RemoveNode(nodeID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	if _, exists := nm.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}
	delete(nm.nodes, nodeID)
	delete(nm.clients, nodeID)
	return nil
}

func (nm *NodeManager) GetNode(nodeID string) (*Node, bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	node, ok := nm.nodes[nodeID]
	return node, ok
}

func (nm *NodeManager) SetNodePaths(nodeID, localDir, remoteDir string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	node, ok := nm.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	node.LocalDir = localDir
	node.RemoteDir = remoteDir
	return nil
}

func (nm *NodeManager) GetClient(nodeID string) (AgentClient, bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	client, ok := nm.clients[nodeID]
	return client, ok
}

func (nm *NodeManager) AllNodes() []*Node {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	nodes := make([]*Node, 0, len(nm.nodes))
	for _, n := range nm.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (nm *NodeManager) StartHeartbeat() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-nm.stopCh:
				return
			case <-ticker.C:
				nm.heartbeatAll()
			}
		}
	}()
}

func (nm *NodeManager) heartbeatAll() {
	nm.mu.RLock()
	ids := make([]string, 0, len(nm.nodes))
	for id := range nm.nodes {
		ids = append(ids, id)
	}
	nm.mu.RUnlock()

	for _, id := range ids {
		nm.mu.RLock()
		client, ok := nm.clients[id]
		node, nodeOk := nm.nodes[id]
		nm.mu.RUnlock()
		if !ok || !nodeOk {
			continue
		}
		err := client.Ping()
		nm.mu.Lock()
		if err != nil {
			if node.Status == NodeConnected {
				node.Status = NodeDisconnected
			}
		} else if node.Status == NodeDisconnected {
			node.Status = NodeConnected
		}
		nm.mu.Unlock()
	}
}

func (nm *NodeManager) Stop() {
	select {
	case <-nm.stopCh:
	default:
		close(nm.stopCh)
	}
}