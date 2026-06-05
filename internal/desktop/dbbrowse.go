package desktop

// Postgres browse bridge. Wails-bound methods the frontend calls.
// Remote nodes go through the deployed agent's /pg endpoints; local mode calls
// the same agentserver functions in-process (mirrors GetLocalMonitor).
//
// To add a browse feature: add a func in agentserver, a method on AgentClient,
// then one wrapper here + a UI call.

import (
	"context"
	"time"

	"github.com/john221wick/postgresMonitor/internal/agentserver"
	"github.com/john221wick/postgresMonitor/internal/cluster"
)

// remoteClient returns the agent client for a remote node, or false for local.
func (a *App) remoteClient(nodeID string) (cluster.AgentClient, bool) {
	if a.manager == nil || nodeID == "" || nodeID == "local" {
		return nil, false
	}
	return a.manager.GetClient(nodeID)
}

// PgDatabases lists databases on the node's Postgres (also acts as connect test).
func (a *App) PgDatabases(nodeID string, req agentserver.PgConnReq) ([]string, error) {
	if client, ok := a.remoteClient(nodeID); ok {
		return client.PgDatabases(req)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return agentserver.PgListDatabases(ctx, req)
}

// PgTables lists user tables in req.DB.
func (a *App) PgTables(nodeID string, req agentserver.PgConnReq) ([]agentserver.PgTable, error) {
	if client, ok := a.remoteClient(nodeID); ok {
		return client.PgTables(req)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return agentserver.PgListTables(ctx, req)
}

// PgRows returns a page of rows from req.Schema.req.Table.
func (a *App) PgRows(nodeID string, req agentserver.PgRowsReq) (agentserver.PgPage, error) {
	if client, ok := a.remoteClient(nodeID); ok {
		return client.PgRows(req)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return agentserver.PgQueryRows(ctx, req)
}

// PgDeleteRow deletes one row (by ctid). Returns rows affected.
func (a *App) PgDeleteRow(nodeID string, req agentserver.PgDeleteReq) (int64, error) {
	if client, ok := a.remoteClient(nodeID); ok {
		return client.PgDeleteRow(req)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return agentserver.PgDeleteRow(ctx, req)
}

// PgInsertRow inserts one row. Returns rows affected.
func (a *App) PgInsertRow(nodeID string, req agentserver.PgInsertReq) (int64, error) {
	if client, ok := a.remoteClient(nodeID); ok {
		return client.PgInsertRow(req)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return agentserver.PgInsertRow(ctx, req)
}

// PgUpdateCell sets one column of one row (by ctid). Returns rows affected.
func (a *App) PgUpdateCell(nodeID string, req agentserver.PgUpdateReq) (int64, error) {
	if client, ok := a.remoteClient(nodeID); ok {
		return client.PgUpdateCell(req)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return agentserver.PgUpdateCell(ctx, req)
}
