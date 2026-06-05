# Goal of this project

To make a postgres db monitor with ui. (Desktop GUI app for monitoring postgres db)

I have postgres server running in my pc or somewhere in remote vps, i have difficulty monitoring it, like there is database, tables, many connections, queries running, cpu/memory usage and many more things. I want to make it simple by just maintaining it via desktop GUI. 

The idea is simple: I can just remote `ssh system` and it will get into that machine. Once it gets to machine it will auto upload a remote binary which will have http endpoints defined, so the local ui will send signals through ssh tunnel to these http endpoints, and it will monitor those things from there. In short it is pretty simple, use golang to send the binaries and execute bunch of commands to get the info about the machine itself, and postgres ofcourse.

Let me explain now is how everything is actually done. For system monitoring, right now i am mostly using linux `/proc`, like `/proc/stat` for cpu, `/proc/meminfo` for memory, `/proc/loadavg`, `/proc/uptime`, `/proc/cpuinfo` and kernel info also from proc/sys. For process list i am using `ps`, and for docker containers i am using docker cli itself like `docker ps` and `docker stats --no-stream`. So this is very linux first right now, which is okay because most postgres servers i care about will be linux servers only.

For postgres side i am using `pgx`, and the app asks postgres itself for databases, tables and rows. For rows i am doing paginated queries, and for update/delete right now i am using `ctid`(which i still have to understand deeply) because it was the simplest way to target a row without first building primary-key detection. I know this is not the final thing, later i should use primary key when table has one and fallback to `ctid` only when there is nothing better.

For remote machine part, there is no big magic. Desktop connects with ssh, detects remote os/arch, cross compiles the small agent binary, uploads it to the remote machine, starts it on `127.0.0.1` there, then creates local port forwarding. So the remote http server is not exposed to internet, it is only reachable through ssh tunnel. The frontend is Svelte, but it calls Wails bindings, then Go backend decides if it should run locally or send request to remote agent.

For now i kept it simple, but still some good progress is done. Earlier it was just database, tables, and user can view records. Now user can also connect local or remote postgres, browse databases, see tables, view paginated rows, add row, update one cell, and delete selected rows.

In future i want to monitor everything postgres has to offer, like if user want to see the speed of queries, cache miss, slow queries, connections, locks, indexes, vacuum, replication and many different things postgres has to offer, we can just use postgres monitor to monitor those. Earlier i thought about using ebpf for monitoring it to base level, but postgres 18 gave many option and it became a liability rather than asset to maintain ebpf, so i decided to switch to postgres one instead.

Yes, I am not an expert in postgres, that is one thing, i will constantly try to learn along this journey. Almost everything from **first principles**, I have build a db from scratch once (yet to be uploaded on github because it is not complete yet), although it was basic one, but it gave me lots of insight on how we can build this from scratch.

I want to also mention that i dont like abstractions, it just blocks my brain for no reason, and i don't have to do manual things now like with the help of ai, mostly the frontend and wails part ai will handle, for me frontend is difficult i have to mention it, people say its easy, i genuinely believe its difficult atleast for me but since the advent of ai, i can just focus on core principles. Also thanks to **Ben Dickens**, what an amazing teacher he is. His db course is literally the best course on db i have seen till now.

# Current progress

This is not just a readme idea now, there is actual app structure in place.

- Desktop app is made with Go + Wails + Svelte.
- There is a local mode and remote mode.
- Local mode can collect system stats directly from the machine.
- Remote mode can connect through ssh, detect remote machine, build linux agent for that arch, upload it, start it remotely, and then talk to it through local port forwarding.
- Remote agent only binds to `127.0.0.1`, so it is not open on public internet. Desktop talks to it through ssh tunnel.
- Saved nodes are persisted in `~/.pgmonitor/desktop-config.json`, so remote machines can be reconnected later.
- There is dashboard, monitor page, database page, terminal page, and settings page.

# Features shipped

## Database browser

- Connect to postgres using host, port, user, password.
- Works for local postgres and also postgres running on a remote node.
- Lists all non-template databases.
- Lists user tables from non-system schemas.
- Shows estimated row count for tables, and for new/not analyzed tables it tries bounded exact count so ui does not show wrong `0`.
- Shows table rows with pagination.
- Shows column names and postgres types.
- Handles `NULL` values properly in ui.
- Recent database connections are saved in browser localStorage, but password is not saved.
- User can add a row.
- User can double click and update one cell.
- User can select rows and delete them.
- Inputs are type aware for basic types like number, boolean, date, datetime.

## Monitor

- Shows host cpu, memory, load average, uptime, os, kernel, arch, cpu model.
- Shows per-core cpu stats on linux.
- Shows top processes sorted by cpu or memory.
- Shows running docker containers if docker is available.
- Shows container name, image, status, cpu and memory.
- Manual refresh is there, and auto refresh every 5 seconds is optional because docker stats can add latency.

## Remote nodes

- User can paste ssh command like `ssh -p 20544 root@host`.
- SSH config alias like `ssh system` is also supported.
- Optional ssh key path can be passed.
- App detects remote hostname, os and arch.
- App cross-compiles the agent for linux `amd64` or `arm64`.
- App uploads the binary to `~/postgresmonitor`.
- App starts remote agent on a free remote port.
- App opens local port forward to the remote agent.
- App has heartbeat(basically pooling sort of thing) which marks node connected/disconnected.
- User can disconnect, remove, or reconnect saved nodes.
- User can change remote destination path.
- User can sync local files to remote node using rsync.

## Terminal

- Remote mode has an embedded xterm terminal.
- It opens an interactive ssh shell with pty.
- Terminal resize is forwarded to remote pty.
- Terminal session survives navigation inside app, so switching page does not immediately kill scrollback.

## Desktop ui

- Sidebar navigation is there.
- Dark/light theme toggle is there.
- Command palette search is there.
- Sidebar collapse is there.
- Dashboard changes based on local mode and remote mode.
- Settings page has theme and path preference.

# How it works

In local mode, desktop app calls Go functions directly and gets monitor/database data from the same machine.

In remote mode, desktop app does this:

1. Parse ssh command.
2. Connect to the remote machine through ssh.
3. Detect remote os and arch.
4. Build the agent binary for that linux arch.
5. Upload the agent to remote machine.
6. Start agent on remote loopback.
7. Forward that remote port to local machine.
8. Frontend keeps calling Go methods, and Go either handles it locally or forwards request to remote agent.

So frontend does not need to know too much. It just calls Wails bindings and backend decides whether it is local or remote.

# Things missed / things still rough

These are the parts i know are missing or not good enough yet(There are lot many i missed it here too ig).

- Postgres monitoring is still very basic. I still need `pg_stat_activity`, `pg_stat_statements`, locks, slow queries, cache hit/miss, index usage, bloat, vacuum, checkpoints, replication, wal, connection pool stats etc.
- Database editing currently uses `ctid` to target rows. It works for basic browsing/editing, but it is not proper long term solution. I should use primary key when table has primary key, and only fall back to `ctid`.
- There is no query editor yet. User cannot run custom sql from app.
- There is no explain/analyze view yet.
- There is no graph/history storage yet. Monitor shows current snapshot, but not long term trend.
- Local host stats are best on linux because they use `/proc`. On mac/windows it is partial for now.
- Remote agent is linux focused right now.
- There is no role/permission model inside app.
- Delete/update needs better safety. Right now there is confirm modal, but no undo.
- Docker monitoring is basic, no logs or container actions yet.
- Tests are present for some cluster pieces, but app still needs more integration tests with real postgres and remote agent.

# Tech stack

- Go for backend and agent.
- Wails for desktop app bridge.
- Svelte 5 for frontend.
- Tailwind for styling.
- pgx for postgres.
- xterm.js for terminal.
- ssh + rsync for remote machine flow.

# Final note

This project is still in early stage. I am building it because I want a simple postgres monitor which actually helps me understand what is happening inside my database and machine, without hiding everything behind too much abstraction.

I am learning postgres deeply while building this, so the plan is not to just make a fancy ui. The plan is to understand what postgres exposes, then expose it in a way which is simple enough that i can use it every day.
