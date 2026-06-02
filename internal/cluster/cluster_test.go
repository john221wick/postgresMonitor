package cluster

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/john221wick/postgresMonitor/internal/agentserver"
)

func startTestAgent(t *testing.T) (string, func()) {
	t.Helper()

	srv, err := agentserver.NewAgentServer("")
	if err != nil {
		t.Fatalf("NewAgentServer: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	go http.Serve(ln, srv.Handler())

	for i := 0; i < 20; i++ {
		if err := NewAgentClient(url).Ping(); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return url, func() { ln.Close() }
}

func TestParseSSHCommand(t *testing.T) {
	tests := []struct {
		input   string
		user    string
		host    string
		port    int
		wantErr bool
	}{
		{"ssh -p 20544 root@203.0.113.10", "root", "203.0.113.10", 20544, false},
		{"ssh root@192.168.1.100", "root", "192.168.1.100", 22, false},
		{"ssh -p 41922 root@ssh5.vast.ai -L 8080:localhost:8080", "root", "ssh5.vast.ai", 41922, false},
		{"-p 9999 user@example.com", "user", "example.com", 9999, false},
		{"ssh -i /tmp/key -p 2222 ubuntu@10.0.0.1", "ubuntu", "10.0.0.1", 2222, false},
		{"", "", "", 0, true},
		{"ssh", "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg, err := ParseSSHCommand(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.User != tt.user {
				t.Errorf("user: got %q, want %q", cfg.User, tt.user)
			}
			if cfg.Host != tt.host {
				t.Errorf("host: got %q, want %q", cfg.Host, tt.host)
			}
			if cfg.Port != tt.port {
				t.Errorf("port: got %d, want %d", cfg.Port, tt.port)
			}
		})
	}
}

func TestParseSSHCommandAlias(t *testing.T) {
	cfg, err := ParseSSHCommand("ssh myalias")
	if err != nil {
		t.Fatalf("bare alias should not error: %v", err)
	}
	if cfg.Host == "" {
		t.Error("host should not be empty")
	}
	t.Logf("alias resolved: host=%s user=%s port=%d key=%s", cfg.Host, cfg.User, cfg.Port, cfg.KeyPath)
}

func TestAgentClientMonitor(t *testing.T) {
	url, cleanup := startTestAgent(t)
	defer cleanup()

	client := NewAgentClient(url)
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	mon, err := client.GetMonitor()
	if err != nil {
		t.Fatalf("GetMonitor: %v", err)
	}
	if mon.CollectedAt == "" {
		t.Error("expected collectedAt timestamp")
	}
}

func TestNodeManager(t *testing.T) {
	url, cleanup := startTestAgent(t)
	defer cleanup()

	mgr := NewNodeManager()
	client := NewAgentClient(url)

	node, err := mgr.AddRemoteNode("node-a", "test-node", client)
	if err != nil {
		t.Fatalf("AddRemoteNode: %v", err)
	}
	if node.Status != NodeConnected {
		t.Fatalf("expected connected, got %s", node.Status)
	}

	nodes := mgr.AllNodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	if err := mgr.RemoveNode("node-a"); err != nil {
		t.Fatalf("RemoveNode: %v", err)
	}
	if len(mgr.AllNodes()) != 0 {
		t.Fatal("expected no nodes after remove")
	}
}