//go:build integration

package cluster

import (
	"fmt"
	"os"
	"testing"
)

// Run with: go test -tags integration -v -timeout 120s ./internal/cluster/ -run TestSSHIntegration
func TestSSHIntegration(t *testing.T) {
	config := &SSHConfig{
		User:    "root",
		Host:    "203.0.113.10",
		Port:    22,
		KeyPath: os.Getenv("HOME") + "/.ssh/host",
	}

	t.Logf("Connecting to %s@%s:%d...", config.User, config.Host, config.Port)
	session, err := Connect(*config)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()
	t.Log("SSH connected!")

	out, err := session.RunCommand("hostname && uname -m")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	t.Logf("Remote: %s", out)

	remoteBase := "~/postgresmonitor"
	if err := session.CrossCompileAndSCP(remoteBase + "/pgmonitor"); err != nil {
		t.Fatalf("CrossCompileAndSCP failed: %v", err)
	}

	if err := session.StartRemoteAgent(9712, remoteBase); err != nil {
		t.Fatalf("StartRemoteAgent failed: %v", err)
	}

	localPort, err := session.ForwardPort(9712)
	if err != nil {
		t.Fatalf("ForwardPort failed: %v", err)
	}
	t.Logf("Port forward: localhost:%d -> remote:9712", localPort)

	client := NewAgentClient(fmt.Sprintf("http://localhost:%d", localPort))
	mon, err := client.GetMonitor()
	if err != nil {
		t.Fatalf("GetMonitor failed: %v", err)
	}
	t.Logf("Remote host: %s CPU=%.1f%% mem=%d/%d MB",
		mon.Host.Hostname, mon.Host.CPUPercent, mon.Host.MemUsedMB, mon.Host.MemTotalMB)

	mgr := NewNodeManager()
	node, err := mgr.AddRemoteNode("ssh-test", config.Host, client)
	if err != nil {
		t.Fatalf("AddRemoteNode failed: %v", err)
	}
	t.Logf("Node added: %s (%s)", node.Name, node.Status)
}