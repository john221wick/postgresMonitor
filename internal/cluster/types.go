package cluster

type NodeStatus int

const (
	NodeConnected NodeStatus = iota
	NodeDisconnected
	NodeConnecting
)

func (s NodeStatus) String() string {
	switch s {
	case NodeConnected:
		return "connected"
	case NodeDisconnected:
		return "disconnected"
	case NodeConnecting:
		return "connecting"
	default:
		return "unknown"
	}
}

type Node struct {
	ID        string
	Name      string
	AgentURL  string
	Status    NodeStatus
	LocalDir  string
	RemoteDir string
	Arch      string
	OS        string
}