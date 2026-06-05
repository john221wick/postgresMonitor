package cluster

import (
	"bytes"
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

	PgDatabases(req agentserver.PgConnReq) ([]string, error)
	PgTables(req agentserver.PgConnReq) ([]agentserver.PgTable, error)
	PgRows(req agentserver.PgRowsReq) (agentserver.PgPage, error)
	PgDeleteRow(req agentserver.PgDeleteReq) (int64, error)
	PgInsertRow(req agentserver.PgInsertReq) (int64, error)
	PgUpdateCell(req agentserver.PgUpdateReq) (int64, error)
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

// postJSON sends body as JSON to path and decodes the response into out.
func (c *httpAgentClient) postJSON(path string, body, out interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.readError(resp)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *httpAgentClient) PgDatabases(req agentserver.PgConnReq) ([]string, error) {
	var out []string
	err := c.postJSON("/pg/databases", req, &out)
	return out, err
}

func (c *httpAgentClient) PgTables(req agentserver.PgConnReq) ([]agentserver.PgTable, error) {
	var out []agentserver.PgTable
	err := c.postJSON("/pg/tables", req, &out)
	return out, err
}

func (c *httpAgentClient) PgRows(req agentserver.PgRowsReq) (agentserver.PgPage, error) {
	var out agentserver.PgPage
	err := c.postJSON("/pg/rows", req, &out)
	return out, err
}

func (c *httpAgentClient) PgDeleteRow(req agentserver.PgDeleteReq) (int64, error) {
	var out struct {
		Affected int64 `json:"affected"`
	}
	err := c.postJSON("/pg/delete", req, &out)
	return out.Affected, err
}

func (c *httpAgentClient) PgInsertRow(req agentserver.PgInsertReq) (int64, error) {
	var out struct {
		Affected int64 `json:"affected"`
	}
	err := c.postJSON("/pg/insert", req, &out)
	return out.Affected, err
}

func (c *httpAgentClient) PgUpdateCell(req agentserver.PgUpdateReq) (int64, error) {
	var out struct {
		Affected int64 `json:"affected"`
	}
	err := c.postJSON("/pg/update", req, &out)
	return out.Affected, err
}

func (c *httpAgentClient) readError(resp *http.Response) error {
	var errResp agentserver.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("agent error (HTTP %d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("agent error: HTTP %d", resp.StatusCode)
}