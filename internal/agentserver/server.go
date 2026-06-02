package agentserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

type AgentServer struct {
	httpServer *http.Server
	listener   net.Listener
	dataDir    string
}

// NewAgentServer creates a lightweight monitor agent (host + Docker).
func NewAgentServer(dir string) (*AgentServer, error) {
	dataDir := dir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dataDir = filepath.Join(home, "postgresmonitor")
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0755); err != nil {
		return nil, fmt.Errorf("create dirs: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /monitor", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, CollectMonitor())
	})
	mux.HandleFunc("GET /topology", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, TopologyResponse{OK: true})
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, StatusResponse{Jobs: []JobStatusResponse{}})
	})

	return &AgentServer{
		httpServer: &http.Server{Handler: mux},
		dataDir:    dataDir,
	}, nil
}

func (s *AgentServer) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = ln
	fmt.Printf("Monitor agent listening on %s\n", addr)
	return s.httpServer.Serve(ln)
}

func (s *AgentServer) Handler() http.Handler {
	return s.httpServer.Handler
}

func (s *AgentServer) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}