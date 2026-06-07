# Postgres Monitor

Desktop GUI app to monitor and manage PostgreSQL — local or over SSH. Connects to a remote box over SSH, ships a small loopback-only agent, and talks to it through an SSH tunnel (agent never exposed to the internet).

## What it does

**Database browser**
- Connect to postgres by host, port, user, password — local or on a remote node.
- List all non-template databases.
- List user tables from non-system schemas.
- Show estimated row counts (bounded exact count for new/unanalyzed tables so it isn't a wrong `0`).
- Browse rows with pagination, column names, and postgres types.
- Handle `NULL` values correctly.
- Add a row.
- Double-click to update a single cell.
- Select rows and delete them.
- Type-aware inputs for number, boolean, date, datetime.
- Recent connections saved in localStorage (password never saved).

**Monitor**
- Host CPU, memory, load average, uptime, OS, kernel, arch, CPU model.
- Per-core CPU stats on Linux.
- Top processes sorted by CPU or memory.
- Running docker containers (name, image, status, CPU, memory) when docker is available.
- Manual refresh, plus optional 5s auto-refresh.

**Remote nodes (over SSH)**
- Paste an ssh command (`ssh -p 20544 root@host`) or use an ssh config alias (`ssh system`).
- Optional ssh key path.
- Detect remote hostname, OS, arch.
- Cross-compile the agent for linux amd64/arm64, upload it, start it on a free remote port.
- Local port-forward to the agent; loopback-only, reachable only through the tunnel.
- Heartbeat marks nodes connected/disconnected.
- Disconnect, remove, or reconnect saved nodes.
- Change remote destination path.
- Sync local files to a remote node via rsync.
- Saved nodes persisted in `~/.pgmonitor/desktop-config.json`.

**Terminal**
- Embedded xterm terminal in remote mode.
- Interactive ssh shell with pty; resize forwarded to the remote pty.
- Session survives in-app navigation (scrollback kept).

**Desktop app**
- Local mode and remote mode.
- Sidebar nav with collapse, command-palette search.
- Dashboard, monitor, database, terminal, settings pages.
- Dark/light theme toggle.
- Settings: theme, path preference, app update, uninstall.

## Install

macOS and Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/john221wick/postgresMonitor/main/install.sh | bash
```

Installs the latest desktop app.

## Docs

Full write-up — architecture, what's shipped, what's rough, tech stack: see [Docs.md](./Docs.md).
