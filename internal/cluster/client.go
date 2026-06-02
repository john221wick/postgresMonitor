package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/john221wick/postgresMonitor/internal/agentserver"
)

type AgentClient interface {
	GetMonitor() (*agentserver.MonitorResponse, error)
	Ping() error
}

type httpAgentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAgentClient(baseURL string) AgentClient {
	return &httpAgentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewUnixAgentClient(socketPath string) AgentClient {
	return &httpAgentClient{
		baseURL: "http://unix",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

func (c *httpAgentClient) GetMonitor() (*agentserver.MonitorResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/monitor")
	if err != nil {
		return nil, fmt.Errorf("get monitor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.readError(resp)
	}

	var result agentserver.MonitorResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode monitor: %w", err)
	}
	return &result, nil
}

func (c *httpAgentClient) Ping() error {
	resp, err := c.httpClient.Get(c.baseURL + "/monitor")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *httpAgentClient) readError(resp *http.Response) error {
	var errResp agentserver.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("agent error (HTTP %d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("agent error: HTTP %d", resp.StatusCode)
}