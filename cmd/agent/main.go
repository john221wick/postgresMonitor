// Remote monitor agent: host stats + Docker. Started on the server via:
//
//	pgmonitor --agent --port 9712 --dir ~/postgresmonitor
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/john221wick/postgresMonitor/internal/agentserver"
)

func main() {
	port := 9712
	dataDir := ""

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--agent":
		case "--port":
			if i+1 < len(os.Args) {
				if n, err := strconv.Atoi(os.Args[i+1]); err == nil {
					port = n
				}
				i++
			}
		case "--dir":
			if i+1 < len(os.Args) {
				dataDir = os.Args[i+1]
				i++
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown argument: %s\n", os.Args[i])
			os.Exit(2)
		}
	}

	srv, err := agentserver.NewAgentServer(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent start failed: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		os.Exit(0)
	}()

	// Bind loopback only: the desktop reaches the agent through an SSH tunnel
	// (which dials the remote's localhost), so there is no reason to expose
	// these endpoints — especially the DB read/write ones — on a public interface.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if err := srv.ListenAndServe(addr); err != nil {
		fmt.Fprintf(os.Stderr, "agent server error: %v\n", err)
		os.Exit(1)
	}
}