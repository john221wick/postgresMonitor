package agentserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

	// --- Postgres browse (read-only) ---
	mux.HandleFunc("POST /pg/databases", func(w http.ResponseWriter, r *http.Request) {
		var req PgConnReq
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		dbs, err := PgListDatabases(ctx, req)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, dbs)
	})
	mux.HandleFunc("POST /pg/tables", func(w http.ResponseWriter, r *http.Request) {
		var req PgConnReq
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		tables, err := PgListTables(ctx, req)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, tables)
	})
	mux.HandleFunc("POST /pg/rows", func(w http.ResponseWriter, r *http.Request) {
		var req PgRowsReq
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		page, err := PgQueryRows(ctx, req)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, page)
	})
	mux.HandleFunc("POST /pg/delete", func(w http.ResponseWriter, r *http.Request) {
		var req PgDeleteReq
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		n, err := PgDeleteRow(ctx, req)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"affected": n})
	})
	mux.HandleFunc("POST /pg/insert", func(w http.ResponseWriter, r *http.Request) {
		var req PgInsertReq
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		n, err := PgInsertRow(ctx, req)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"affected": n})
	})
	mux.HandleFunc("POST /pg/update", func(w http.ResponseWriter, r *http.Request) {
		var req PgUpdateReq
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		n, err := PgUpdateCell(ctx, req)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"affected": n})
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