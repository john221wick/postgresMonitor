/*
 * API client — wraps Wails Go bindings.
 * Bindings are generated at frontend/wailsjs/go/desktop/App.js during `wails dev` / `wails build`.
 */

import {
	SetRemoteMode,
	GetRemoteMode,
	ConnectNode,
	DisconnectNode,
	GetNodes,
	GetSavedNodes,
	ReconnectNode,
	RemoveNode,
	SetNodePaths,
	SyncFilesToNode,
	StartTerminalSession,
	WriteTerminalInput,
	ResizeTerminal,
	StopTerminalSession,
	GetClusterMonitor,
	GetLocalMonitor,
	PgDatabases,
	PgTables,
	PgRows,
	PgDeleteRow,
	PgInsertRow,
	PgUpdateCell,
	GetAppVersion,
	CheckAppUpdate,
	DownloadAndApplyUpdate,
	UninstallDesktopApp
} from '../../wailsjs/go/desktop/App.js';

export {
	SetRemoteMode,
	GetRemoteMode,
	ConnectNode,
	DisconnectNode,
	GetNodes,
	GetSavedNodes,
	ReconnectNode,
	RemoveNode,
	SetNodePaths,
	SyncFilesToNode,
	StartTerminalSession,
	WriteTerminalInput,
	ResizeTerminal,
	StopTerminalSession,
	GetClusterMonitor,
	GetLocalMonitor,
	PgDatabases,
	PgTables,
	PgRows,
	PgDeleteRow,
	PgInsertRow,
	PgUpdateCell,
	// App update / uninstall
	GetAppVersion,
	CheckAppUpdate,
	DownloadAndApplyUpdate,
	UninstallDesktopApp
};