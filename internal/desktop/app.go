package desktop

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/john221wick/postgresMonitor/internal/agentserver"
	"github.com/john221wick/postgresMonitor/internal/cluster"
)

// App is the Wails application bridge.
type App struct {
	ctx         context.Context
	remoteMode  bool
	manager     *cluster.NodeManager
	sshSessions map[string]*cluster.SSHSession
	savedNodes  map[string]*SavedNode
}

func NewApp() *App {
	return &App{
		sshSessions: make(map[string]*cluster.SSHSession),
		savedNodes:  make(map[string]*SavedNode),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.loadSavedConfig()
}

func (a *App) Shutdown(ctx context.Context) {
	if a.manager != nil {
		a.manager.Stop()
	}
	for _, sess := range a.sshSessions {
		sess.Close()
	}
}

func (a *App) loadSavedConfig() {
	cfg, err := LoadDesktopConfig()
	if err != nil {
		return
	}
	for _, sn := range cfg.Nodes {
		snCopy := sn
		a.savedNodes[sn.ID] = &snCopy
	}
	if cfg.RemoteMode {
		a.SetRemoteMode(true)
	}
}

func (a *App) saveConfig() {
	cfg := &DesktopConfig{RemoteMode: a.remoteMode}
	for _, sn := range a.savedNodes {
		if a.manager != nil {
			if node, ok := a.manager.GetNode(sn.ID); ok {
				sn.LocalDir = node.LocalDir
				sn.RemoteDir = node.RemoteDir
			}
		}
		cfg.Nodes = append(cfg.Nodes, *sn)
	}
	_ = SaveDesktopConfig(cfg)
}

// --- Mode ---

func (a *App) SetRemoteMode(enabled bool) error {
	if enabled && a.manager == nil {
		a.manager = cluster.NewNodeManager()
		a.manager.StartHeartbeat()
	}
	a.remoteMode = enabled
	a.saveConfig()
	return nil
}

func (a *App) GetRemoteMode() bool {
	return a.remoteMode
}

// --- DTOs ---

type NodeInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	LocalDir        string `json:"localDir"`
	RemoteDir       string `json:"remoteDir"`
	Arch string `json:"arch"`
	OS   string `json:"os"`
}

type NodeMonitorInfo struct {
	NodeID      string                   `json:"nodeID"`
	NodeName    string                   `json:"nodeName"`
	Reachable   bool                     `json:"reachable"`
	Error       string                   `json:"error,omitempty"`
	Host        agentserver.HostStats       `json:"host"`
	Containers  agentserver.ContainerReport `json:"containers"`
	Processes   []agentserver.ProcInfo   `json:"processes"`
	CollectedAt string                   `json:"collectedAt"`
}

type SavedNodeInfo struct {
	ID         string `json:"id"`
	SSHCommand string `json:"sshCommand"`
}

// --- Nodes ---

func (a *App) ConnectNode(sshCommand string, keyPath string) (NodeInfo, error) {
	if a.manager == nil {
		return NodeInfo{}, fmt.Errorf("remote mode not enabled")
	}

	config, err := cluster.ParseSSHCommand(sshCommand)
	if err != nil {
		return NodeInfo{}, fmt.Errorf("invalid SSH command: %w", err)
	}
	if keyPath != "" {
		config.KeyPath = keyPath
	}

	session, err := cluster.Connect(*config)
	if err != nil {
		return NodeInfo{}, fmt.Errorf("SSH connection failed: %w", err)
	}

	remoteInfo, err := session.DetectRemote()
	if err != nil {
		session.Close()
		return NodeInfo{}, fmt.Errorf("detection failed: %w", err)
	}

	remoteBase := "~/postgresmonitor"
	session.RunCommand(fmt.Sprintf("mkdir -p %s/logs", remoteBase))

	if err := session.CrossCompileAndSCP(remoteBase + "/pgmonitor"); err != nil {
		session.Close()
		return NodeInfo{}, fmt.Errorf("deploy agent failed: %w", err)
	}

	// Pick a free port on the remote to avoid colliding with existing services.
	agentPort, err := session.FreeRemotePort(20000, 60000)
	if err != nil {
		agentPort = 47215 // fallback: uncommon fixed port
	}

	if err := session.StartRemoteAgent(agentPort, remoteBase); err != nil {
		session.Close()
		return NodeInfo{}, fmt.Errorf("start remote agent failed: %w", err)
	}

	localPort, err := session.ForwardPort(agentPort)
	if err != nil {
		session.Close()
		return NodeInfo{}, fmt.Errorf("port forward failed: %w", err)
	}

	client := cluster.NewAgentClient(fmt.Sprintf("http://localhost:%d", localPort))
	nodeID := fmt.Sprintf("ssh-%s-%d", config.Host, config.Port)

	nodeName := remoteInfo.Hostname
	if nodeName == "" {
		nodeName = config.Host
	}

	node, err := a.manager.AddRemoteNode(nodeID, nodeName, client)
	if err != nil {
		session.Close()
		return NodeInfo{}, fmt.Errorf("add node failed: %w", err)
	}

	node.Arch = remoteInfo.Arch
	node.OS = remoteInfo.OS
	if absOut, err := session.RunCommand(fmt.Sprintf("eval echo %s", remoteBase)); err == nil {
		if absPath := strings.TrimSpace(absOut); absPath != "" {
			node.RemoteDir = absPath
		}
	}
	if node.RemoteDir == "" {
		node.RemoteDir = "/root/postgresmonitor"
	}

	a.sshSessions[nodeID] = session
	a.savedNodes[nodeID] = &SavedNode{
		ID:         nodeID,
		SSHCommand: sshCommand,
		KeyPath:    keyPath,
	}
	a.saveConfig()

	return nodeToInfo(node), nil
}

func (a *App) DisconnectNode(nodeID string) error {
	if a.manager == nil {
		return fmt.Errorf("remote mode not enabled")
	}
	if sess, ok := a.sshSessions[nodeID]; ok {
		sess.StopRemoteAgent(a.remoteBaseFor(nodeID))
		sess.Close()
		delete(a.sshSessions, nodeID)
	}
	a.manager.RemoveNode(nodeID)
	a.saveConfig()
	return nil
}

// remoteBaseFor returns the remote install dir for a node, defaulting to the
// install path used at connect time.
func (a *App) remoteBaseFor(nodeID string) string {
	if a.manager != nil {
		if node, ok := a.manager.GetNode(nodeID); ok && node.RemoteDir != "" {
			return node.RemoteDir
		}
	}
	return "~/postgresmonitor"
}

func (a *App) RemoveNode(nodeID string) error {
	if sess, ok := a.sshSessions[nodeID]; ok {
		sess.StopRemoteAgent(a.remoteBaseFor(nodeID))
		sess.Close()
		delete(a.sshSessions, nodeID)
	}
	delete(a.savedNodes, nodeID)
	if a.manager != nil {
		a.manager.RemoveNode(nodeID)
	}
	a.saveConfig()
	return nil
}

func (a *App) ReconnectNode(nodeID string) (NodeInfo, error) {
	sn, ok := a.savedNodes[nodeID]
	if !ok {
		return NodeInfo{}, fmt.Errorf("no saved config for node %s", nodeID)
	}
	return a.ConnectNode(sn.SSHCommand, sn.KeyPath)
}

func (a *App) GetSavedNodes() []SavedNodeInfo {
	var result []SavedNodeInfo
	for _, sn := range a.savedNodes {
		if a.manager != nil {
			if _, ok := a.manager.GetNode(sn.ID); ok {
				continue
			}
		}
		result = append(result, SavedNodeInfo{ID: sn.ID, SSHCommand: sn.SSHCommand})
	}
	return result
}

func (a *App) GetNodes() []NodeInfo {
	if a.manager == nil {
		return nil
	}
	nodes := a.manager.AllNodes()
	result := make([]NodeInfo, len(nodes))
	for i, n := range nodes {
		result[i] = nodeToInfo(n)
	}
	return result
}

func nodeToInfo(n *cluster.Node) NodeInfo {
	return NodeInfo{
		ID:        n.ID,
		Name:      n.Name,
		Status:    n.Status.String(),
		LocalDir:  n.LocalDir,
		RemoteDir: n.RemoteDir,
		Arch:      n.Arch,
		OS:        n.OS,
	}
}

func (a *App) SetNodePaths(nodeID, localDir, remoteDir string) error {
	if a.manager == nil {
		return fmt.Errorf("remote mode not enabled")
	}
	if err := a.manager.SetNodePaths(nodeID, localDir, remoteDir); err != nil {
		return err
	}
	a.saveConfig()
	return nil
}

func (a *App) SyncFilesToNode(nodeID string, localPath string) (string, error) {
	if a.manager == nil {
		return "", fmt.Errorf("remote mode not enabled")
	}
	sess, ok := a.sshSessions[nodeID]
	if !ok {
		return "", fmt.Errorf("no SSH session for node %s", nodeID)
	}
	node, ok := a.manager.GetNode(nodeID)
	if !ok {
		return "", fmt.Errorf("node %s not found", nodeID)
	}
	remoteDir := node.RemoteDir
	if remoteDir == "" {
		remoteDir = "/root/postgresmonitor"
	}
	localPath = strings.TrimRight(localPath, "/")
	sess.RunCommand(fmt.Sprintf("mkdir -p %s", remoteDir))
	if err := cluster.RsyncToRemote(localPath, sess.Config, remoteDir); err != nil {
		return "", err
	}
	return remoteDir, nil
}

// --- Monitor ---

func monitorToNodeInfo(nodeID, nodeName string, mon *agentserver.MonitorResponse, err error) NodeMonitorInfo {
	info := NodeMonitorInfo{NodeID: nodeID, NodeName: nodeName}
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Reachable = true
	info.Host = mon.Host
	info.Containers = mon.Containers
	info.Processes = mon.Processes
	info.CollectedAt = mon.CollectedAt
	return info
}

func (a *App) GetClusterMonitor() []NodeMonitorInfo {
	if a.manager == nil {
		return nil
	}
	result := make([]NodeMonitorInfo, 0)
	for _, n := range a.manager.AllNodes() {
		client, ok := a.manager.GetClient(n.ID)
		if !ok {
			result = append(result, NodeMonitorInfo{NodeID: n.ID, NodeName: n.Name, Error: "node not connected"})
			continue
		}
		mon, err := client.GetMonitor()
		result = append(result, monitorToNodeInfo(n.ID, n.Name, mon, err))
	}
	return result
}

func (a *App) GetLocalMonitor() []NodeMonitorInfo {
	mon := agentserver.CollectMonitor()
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}
	return []NodeMonitorInfo{monitorToNodeInfo("local", hostname, &mon, nil)}
}