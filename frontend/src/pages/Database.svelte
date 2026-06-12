<script>
	import { onMount } from 'svelte';
	import { GetNodes, PgDatabases, PgTables, PgRows, PgDeleteRow, PgInsertRow, PgUpdateCell } from '../lib/api.js';

	let { remoteMode = false } = $props();

	// Connection form
	let nodes = $state([]);
	let nodeId = $state('local');
	let host = $state('127.0.0.1');
	let port = $state(5432);
	let user = $state('postgres');
	let password = $state('');
	let passwordEl = $state(null);

	let connected = $state(false);
	let connecting = $state(false);
	let error = $state('');

	// Saved connections (localStorage; password is never stored)
	const LS_KEY = 'pgmonitor.connections';
	let saved = $state([]);

	// Browse state
	let databases = $state([]);
	let selectedDb = $state('');
	let dbOpen = $state(false);
	let tables = $state([]);
	let tablesLoading = $state(false);
	let selectedTable = $state(null);
	let page = $state(null);
	let rowsLoading = $state(false);
	let limit = $state(50);
	let offset = $state(0);
	let sidebarOpen = $state(true);

	// Mutations
	let confirmDel = $state(null); // { ri, ctid }
	let deleting = $state(false);
	let adding = $state(false);
	let newRow = $state({});
	let inserting = $state(false);

	// Edit one cell (popup confirms before applying)
	let editCell = $state(null); // { ri, ci, ctid, column, type, orig }
	let editValue = $state('');
	let editNull = $state(false);
	let updating = $state(false);

	// JSON viewer
	let jsonView = $state(null); // { column, obj }

	// Pick the right editor widget for the column type.
	// Map a Postgres type name to an input widget kind.
	function kindOf(type) {
		const t = (type || '').toLowerCase();
		if (t === 'bool' || t === 'boolean') return 'bool';
		if (['int2', 'int4', 'int8', 'float4', 'float8', 'numeric', 'money', 'oid'].includes(t)) return 'number';
		if (t === 'date') return 'date';
		if (t.startsWith('timestamp')) return 'datetime';
		return 'text';
	}
	// Returns an error message for a bad value, or '' if valid/empty.
	function validateValue(kind, val) {
		const s = String(val).trim();
		if (s === '') return '';
		if (kind === 'number' && !/^-?(\d+\.?\d*|\.\d+)(e-?\d+)?$/i.test(s)) return 'must be a number';
		if (kind === 'bool' && !['true', 'false', 't', 'f', '1', '0'].includes(s.toLowerCase())) return 'must be true or false';
		return '';
	}
	const editKind = $derived(kindOf(editCell?.type || ''));

	const nodeName = $derived(
		remoteMode ? nodes.find((n) => n.id === nodeId)?.name || nodeId || 'node' : 'local'
	);

	function baseReq() {
		return { host: host.trim(), port: Number(port) || 5432, user: user.trim(), password, sslMode: 'disable' };
	}

	// --- saved connections ---
	function loadSaved() {
		try {
			saved = JSON.parse(localStorage.getItem(LS_KEY) || '[]');
		} catch {
			saved = [];
		}
	}
	function persistSaved() {
		try {
			localStorage.setItem(LS_KEY, JSON.stringify(saved));
		} catch {}
	}
	function rememberConnection() {
		const e = { nodeId, nodeName, host: host.trim(), port: Number(port) || 5432, user: user.trim() };
		saved = [e, ...saved.filter((s) => !(s.nodeId === e.nodeId && s.host === e.host && s.port === e.port && s.user === e.user))].slice(0, 8);
		persistSaved();
	}
	function useSaved(s) {
		nodeId = s.nodeId;
		host = s.host;
		port = s.port;
		user = s.user;
		password = '';
		error = '';
		passwordEl?.focus();
	}
	function removeSaved(s, ev) {
		ev.stopPropagation();
		saved = saved.filter((x) => x !== s);
		persistSaved();
	}

	// --- connect / browse ---
	async function connect() {
		error = '';
		connecting = true;
		connected = false;
		databases = [];
		selectedDb = '';
		dbOpen = false;
		tables = [];
		selectedTable = null;
		page = null;
		try {
			databases = (await PgDatabases(nodeId, baseReq())) || [];
			connected = true;
			rememberConnection();
		} catch (e) {
			error = e?.message || String(e);
		} finally {
			connecting = false;
		}
	}

	function disconnect() {
		connected = false;
		databases = [];
		selectedDb = '';
		dbOpen = false;
		tables = [];
		selectedTable = null;
		page = null;
		adding = false;
		confirmDel = null;
		selectedCtids = [];
		error = '';
	}

	async function openDb(db) {
		if (selectedDb === db) {
			dbOpen = !dbOpen;
			return;
		}
		selectedDb = db;
		dbOpen = true;
		selectedTable = null;
		page = null;
		tables = [];
		tablesLoading = true;
		error = '';
		try {
			tables = (await PgTables(nodeId, { ...baseReq(), db })) || [];
		} catch (e) {
			error = e?.message || String(e);
		} finally {
			tablesLoading = false;
		}
	}

	async function openTable(t) {
		selectedTable = t;
		offset = 0;
		adding = false;
		await loadRows();
	}

	async function loadRows() {
		if (!selectedTable) return;
		selectedCtids = [];
		rowsLoading = true;
		error = '';
		try {
			page = await PgRows(nodeId, {
				...baseReq(),
				db: selectedDb,
				schema: selectedTable.schema,
				table: selectedTable.name,
				limit: Number(limit) || 50,
				offset
			});
		} catch (e) {
			error = e?.message || String(e);
			page = null;
		} finally {
			rowsLoading = false;
		}
	}

	function nextPage() {
		if (!page?.hasMore) return;
		offset += Number(limit) || 50;
		loadRows();
	}
	function prevPage() {
		offset = Math.max(0, offset - (Number(limit) || 50));
		loadRows();
	}

	// --- selection + delete ---
	let selectedCtids = $state([]);
	function isSel(ctid) {
		return selectedCtids.includes(ctid);
	}
	function toggleRow(ctid) {
		selectedCtids = isSel(ctid) ? selectedCtids.filter((c) => c !== ctid) : [...selectedCtids, ctid];
	}
	const allSelected = $derived(!!page?.ctids?.length && selectedCtids.length === page.ctids.length);
	function toggleAll() {
		selectedCtids = allSelected ? [] : [...(page?.ctids || [])];
	}
	function askDeleteSelected() {
		if (selectedCtids.length) confirmDel = { ctids: [...selectedCtids] };
	}
	async function doDelete() {
		const ctids = confirmDel?.ctids || [];
		if (!ctids.length) {
			confirmDel = null;
			return;
		}
		deleting = true;
		error = '';
		try {
			for (const ctid of ctids) {
				await PgDeleteRow(nodeId, {
					...baseReq(),
					db: selectedDb,
					schema: selectedTable.schema,
					table: selectedTable.name,
					ctid
				});
			}
			confirmDel = null;
			selectedCtids = [];
			await loadRows();
		} catch (e) {
			error = e?.message || String(e);
		} finally {
			deleting = false;
		}
	}

	// --- insert ---
	function startAdd() {
		newRow = {};
		adding = true;
		error = '';
	}
	async function insertRow() {
		const values = {};
		const cols = page?.columns || [];
		for (let i = 0; i < cols.length; i++) {
			const col = cols[i];
			const v = newRow[col];
			if (v === undefined || v === null || String(v).length === 0) continue;
			const verr = validateValue(kindOf(page.types?.[i] || ''), v);
			if (verr) {
				error = `${col}: ${verr}`;
				return;
			}
			values[col] = String(v);
		}
		if (Object.keys(values).length === 0) {
			error = 'Fill at least one column';
			return;
		}
		inserting = true;
		error = '';
		try {
			await PgInsertRow(nodeId, {
				...baseReq(),
				db: selectedDb,
				schema: selectedTable.schema,
				table: selectedTable.name,
				values
			});
			adding = false;
			newRow = {};
			await loadRows();
		} catch (e) {
			error = e?.message || String(e);
		} finally {
			inserting = false;
		}
	}

	// --- edit cell ---
	function openEditCell(ri, ci) {
		const ctid = page?.ctids?.[ri] || '';
		if (!ctid) {
			error = 'This row has no id (ctid); cannot edit';
			return;
		}
		const orig = page.rows[ri][ci];
		const type = page.types?.[ci] || '';
		editCell = { ri, ci, ctid, column: page.columns[ci], type, orig };
		editNull = orig === null;
		const t = type.toLowerCase();
		if (t === 'bool' || t === 'boolean') editValue = orig === 'true' ? 'true' : 'false';
		else if ((t === 'json' || t === 'jsonb') && orig != null) {
			try {
				editValue = JSON.stringify(JSON.parse(orig), null, 2);
			} catch {
				editValue = orig;
			}
		} else editValue = orig === null ? '' : orig;
	}
	async function doUpdate() {
		if (!editCell) return;
		if (!editNull) {
			const verr = validateValue(editKind, editValue);
			if (verr) {
				error = `${editCell.column}: ${verr}`;
				return;
			}
		}
		updating = true;
		error = '';
		try {
			await PgUpdateCell(nodeId, {
				...baseReq(),
				db: selectedDb,
				schema: selectedTable.schema,
				table: selectedTable.name,
				ctid: editCell.ctid,
				column: editCell.column,
				value: editNull ? null : editValue
			});
			editCell = null;
			await loadRows();
		} catch (e) {
			error = e?.message || String(e);
		} finally {
			updating = false;
		}
	}

	// --- JSON viewer ---
	// Returns the parsed object/array if the cell is JSON, else undefined.
	function jsonCell(cell, type) {
		if (cell == null) return undefined;
		const t = (type || '').toLowerCase();
		const s = String(cell).trim();
		if (t !== 'json' && t !== 'jsonb' && !s.startsWith('{') && !s.startsWith('[')) return undefined;
		try {
			const v = JSON.parse(s);
			if (v && typeof v === 'object') return v;
		} catch {}
		return undefined;
	}
	function openJson(column, obj) {
		jsonView = { column, obj };
	}
	// Pretty-print + light syntax highlighting (HTML-escaped first → safe).
	function hlJson(obj) {
		let s = JSON.stringify(obj, null, 2);
		s = s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
		return s.replace(
			/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g,
			(m) => {
				let cls = 'num';
				if (/^"/.test(m)) cls = /:$/.test(m) ? 'key' : 'str';
				else if (/true|false/.test(m)) cls = 'bool';
				else if (/null/.test(m)) cls = 'nul';
				return `<span class="j-${cls}">${m}</span>`;
			}
		);
	}
	async function copyJson() {
		try {
			await navigator.clipboard.writeText(JSON.stringify(jsonView.obj, null, 2));
		} catch {}
	}

	function onFormKey(e) {
		if (e.key === 'Enter' && !connecting) connect();
	}

	onMount(async () => {
		loadSaved();
		if (remoteMode) {
			try {
				nodes = (await GetNodes()) || [];
				const first = nodes.find((n) => n.id !== 'local');
				if (first) nodeId = first.id;
			} catch {}
		} else {
			nodeId = 'local';
		}
	});
</script>

<div class="h-full flex flex-col p-5 gap-3 min-h-0">
	{#if !connected}
		<!-- Disconnected: title + form + saved connections -->
		<div>
			<h1 class="text-lg font-semibold" style="color: var(--text-primary);">Database</h1>
			<p class="text-[13px] mt-0.5" style="color: var(--text-tertiary);">
				Browse Postgres databases, tables and rows {remoteMode ? '(remote node)' : '(local)'}
			</p>
		</div>

		<div class="rounded-xl p-4 shrink-0" style="background: var(--bg-secondary); border: 1px solid var(--border);">
			<div class="flex flex-wrap items-end gap-3">
				{#if remoteMode}
					<div class="flex flex-col gap-1">
						<span class="text-[10.5px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Node</span>
						<select bind:value={nodeId}
							class="px-2.5 h-9 rounded-md text-[13px] outline-none"
							style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);">
							{#each nodes.filter((n) => n.id !== 'local') as n (n.id)}
								<option value={n.id}>{n.name}</option>
							{:else}
								<option value="">No connected nodes</option>
							{/each}
						</select>
					</div>
				{/if}
				<div class="flex flex-col gap-1">
					<span class="text-[10.5px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Host</span>
					<input bind:value={host} onkeydown={onFormKey} spellcheck="false" autocomplete="off"
						class="w-36 px-2.5 h-9 rounded-md text-[13px] font-[JetBrains_Mono,monospace] outline-none"
						style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
				</div>
				<div class="flex flex-col gap-1">
					<span class="text-[10.5px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Port</span>
					<input bind:value={port} onkeydown={onFormKey} type="number"
						class="w-24 px-2.5 h-9 rounded-md text-[13px] font-[JetBrains_Mono,monospace] outline-none"
						style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
				</div>
				<div class="flex flex-col gap-1">
					<span class="text-[10.5px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">User</span>
					<input bind:value={user} onkeydown={onFormKey} spellcheck="false" autocomplete="off"
						class="w-36 px-2.5 h-9 rounded-md text-[13px] font-[JetBrains_Mono,monospace] outline-none"
						style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
				</div>
				<div class="flex flex-col gap-1">
					<span class="text-[10.5px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Password</span>
					<input bind:this={passwordEl} bind:value={password} onkeydown={onFormKey} type="password" autocomplete="off"
						class="w-40 px-2.5 h-9 rounded-md text-[13px] font-[JetBrains_Mono,monospace] outline-none"
						style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
				</div>
				<button onclick={connect} disabled={connecting}
					class="px-4 h-9 rounded-md text-[13px] font-medium cursor-pointer disabled:opacity-60"
					style="background: var(--accent); color: var(--accent-text);">
					{connecting ? 'Connecting…' : 'Connect'}
				</button>
			</div>
			{#if error}
				<div class="mt-3 rounded-md px-3 py-2 text-[12px] font-[JetBrains_Mono,monospace]" style="background: rgba(239,68,68,0.1); color: rgb(239,68,68);">
					{error}
				</div>
			{/if}
		</div>

		{#if saved.length}
			<div class="shrink-0">
				<div class="text-[10.5px] font-semibold uppercase tracking-wider mb-2" style="color: var(--text-muted);">Recent connections</div>
				<div class="flex flex-col gap-1.5">
					{#each saved as s (s.nodeId + s.host + s.port + s.user)}
						<button onclick={() => useSaved(s)}
							class="group flex items-center gap-2 rounded-lg px-3 h-10 text-left cursor-pointer transition-colors"
							style="background: var(--bg-secondary); border: 1px solid var(--border);">
							<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--text-muted);"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/></svg>
							<span class="text-[12.5px] font-medium" style="color: var(--text-primary);">{s.nodeName}</span>
							<span class="text-[12px] font-[JetBrains_Mono,monospace]" style="color: var(--text-tertiary);">{s.host}:{s.port}</span>
							<span style="color: var(--text-muted);">·</span>
							<span class="text-[12px] font-[JetBrains_Mono,monospace]" style="color: var(--text-tertiary);">{s.user}</span>
							<span class="flex-1"></span>
							<span class="text-[11px]" style="color: var(--text-muted);">click to fill</span>
							<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
							<span role="button" tabindex="-1" onclick={(e) => removeSaved(s, e)} title="Remove"
								class="px-1.5 py-0.5 rounded text-[11px] cursor-pointer" style="color: var(--text-muted);">✕</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}
	{:else}
		<!-- Connected: compact bar + full-height browser -->
		<div class="flex items-center justify-between rounded-lg px-3.5 h-11 shrink-0" style="background: var(--bg-secondary); border: 1px solid var(--border);">
			<div class="flex items-center gap-2 text-[12.5px] min-w-0" style="color: var(--text-secondary);">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--accent);"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/></svg>
				<span class="font-semibold truncate" style="color: var(--text-primary);">{nodeName}</span>
				<span style="color: var(--text-muted);">·</span>
				<span class="font-[JetBrains_Mono,monospace]">{host}:{port}</span>
				<span style="color: var(--text-muted);">·</span>
				<span class="font-[JetBrains_Mono,monospace]">{user}</span>
				<span style="color: var(--text-muted);">·</span>
				<span style="color: var(--text-tertiary);">{databases.length} databases</span>
			</div>
			<button onclick={disconnect}
				class="px-3 h-7 rounded-md text-[12px] font-medium cursor-pointer shrink-0"
				style="background: rgba(239,68,68,0.1); color: rgb(239,68,68);">
				Disconnect
			</button>
		</div>

		{#if error}
			<div class="rounded-md px-3 py-2 text-[12px] font-[JetBrains_Mono,monospace] shrink-0" style="background: rgba(239,68,68,0.1); color: rgb(239,68,68);">
				{error}
			</div>
		{/if}

		<div class="flex-1 min-h-0 flex gap-3">
			{#if sidebarOpen}
			<!-- Sidebar -->
			<div class="w-64 shrink-0 flex flex-col rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border);">
				<div class="flex items-center justify-between px-3 py-2 shrink-0" style="border-bottom: 1px solid var(--border);">
					<span class="text-[10.5px] font-semibold uppercase tracking-wider" style="color: var(--text-muted);">Databases ({databases.length})</span>
					<button onclick={() => (sidebarOpen = false)} title="Hide panel" class="grid place-items-center w-5 h-5 rounded cursor-pointer" style="color: var(--text-muted);">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="11 17 6 12 11 7"/><polyline points="18 17 13 12 18 7"/></svg>
					</button>
				</div>
				<div class="flex-1 min-h-0 overflow-y-auto">
					{#each databases as db (db)}
						{@const isOpen = dbOpen && selectedDb === db}
						<button onclick={() => openDb(db)}
							class="flex items-center gap-2 w-full text-left px-3 py-1.5 text-[12.5px] cursor-pointer transition-colors"
							style="color: {selectedDb === db ? 'var(--text-primary)' : 'var(--text-secondary)'}; background: {isOpen ? 'var(--active-bg)' : 'transparent'};">
							<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" style="color: var(--text-muted); transition: transform .15s; transform: rotate({isOpen ? 90 : 0}deg);"><polyline points="9 18 15 12 9 6"/></svg>
							<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--text-muted);"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/></svg>
							<span class="truncate">{db}</span>
						</button>
						{#if isOpen}
							<div class="pb-1" style="background: var(--bg-tertiary);">
								{#if tablesLoading}
									<div class="px-3 py-2 text-[11.5px]" style="color: var(--text-muted);">Loading tables…</div>
								{:else if tables.length === 0}
									<div class="px-3 py-2 text-[11.5px]" style="color: var(--text-muted);">No tables</div>
								{:else}
									{#each tables as t (t.schema + '.' + t.name)}
										{@const isSel = selectedTable && selectedTable.schema === t.schema && selectedTable.name === t.name}
										<button onclick={() => openTable(t)}
											class="flex items-center justify-between w-full text-left pl-8 pr-3 py-1.5 text-[12px] cursor-pointer transition-colors"
											style="color: {isSel ? 'var(--accent)' : 'var(--text-secondary)'}; background: {isSel ? 'var(--active-bg)' : 'transparent'};">
											<span class="truncate">
												{#if t.schema !== 'public'}<span style="color: var(--text-muted);">{t.schema}.</span>{/if}{t.name}
											</span>
											<span class="text-[10px] font-[JetBrains_Mono,monospace] ml-2 shrink-0" style="color: var(--text-muted);">{t.rows}</span>
										</button>
									{/each}
								{/if}
							</div>
						{/if}
					{/each}
				</div>
			</div>
			{:else}
			<button onclick={() => (sidebarOpen = true)} title="Show databases panel"
				class="shrink-0 w-9 flex flex-col items-center pt-2.5 rounded-xl cursor-pointer transition-colors"
				style="background: var(--bg-secondary); border: 1px solid var(--border); color: var(--text-tertiary);"
				onmouseenter={(e) => (e.currentTarget.style.color = 'var(--text-primary)')}
				onmouseleave={(e) => (e.currentTarget.style.color = 'var(--text-tertiary)')}>
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="13 17 18 12 13 7"/><polyline points="6 17 11 12 6 7"/></svg>
			</button>
			{/if}

			<!-- Main -->
			<div class="flex-1 min-w-0 flex flex-col rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border);">
				{#if !selectedTable}
					<div class="flex items-center justify-center h-full text-[13px]" style="color: var(--text-muted);">
						Pick a database, then a table.
					</div>
				{:else}
					<div class="flex items-center justify-between px-4 py-2.5 shrink-0" style="border-bottom: 1px solid var(--border);">
						<div class="flex items-center gap-2 min-w-0">
							<span class="text-[13px] font-semibold truncate font-[JetBrains_Mono,monospace]" style="color: var(--text-primary);">
								{selectedTable.schema}.{selectedTable.name}
							</span>
							{#if rowsLoading}<span class="text-[11px]" style="color: var(--text-muted);">loading…</span>{/if}
						</div>
						<div class="flex items-center gap-2 shrink-0">
							{#if selectedCtids.length}
								<button onclick={askDeleteSelected}
									class="px-2.5 h-7 rounded text-[12px] font-medium cursor-pointer"
									style="background: rgba(239,68,68,0.12); color: rgb(239,68,68);">Delete ({selectedCtids.length})</button>
							{/if}
							<button onclick={startAdd} disabled={!page?.columns?.length}
								class="px-2.5 h-7 rounded text-[12px] font-medium cursor-pointer disabled:opacity-40"
								style="background: var(--accent); color: var(--accent-text);">+ Add row</button>
							<span class="text-[11px] font-[JetBrains_Mono,monospace]" style="color: var(--text-tertiary);">
								{offset + 1}–{offset + (page?.rows?.length || 0)}
							</span>
							<button onclick={prevPage} disabled={offset === 0 || rowsLoading}
								class="px-2 h-7 rounded text-[12px] font-medium cursor-pointer disabled:opacity-40"
								style="background: var(--bg-tertiary); color: var(--text-secondary);">Prev</button>
							<button onclick={nextPage} disabled={!page?.hasMore || rowsLoading}
								class="px-2 h-7 rounded text-[12px] font-medium cursor-pointer disabled:opacity-40"
								style="background: var(--bg-tertiary); color: var(--text-secondary);">Next</button>
						</div>
					</div>

					{#if adding && page?.columns?.length}
						<div class="px-4 py-3 shrink-0" style="border-bottom: 1px solid var(--border); background: var(--bg-tertiary);">
							<div class="flex items-center justify-between mb-2">
								<span class="text-[11px] font-semibold uppercase tracking-wider" style="color: var(--text-muted);">New row — leave blank for default/NULL</span>
								<div class="flex gap-2">
									<button onclick={insertRow} disabled={inserting}
										class="px-3 h-7 rounded text-[12px] font-medium cursor-pointer disabled:opacity-50"
										style="background: var(--accent); color: var(--accent-text);">{inserting ? 'Inserting…' : 'Insert'}</button>
									<button onclick={() => (adding = false)}
										class="px-3 h-7 rounded text-[12px] font-medium cursor-pointer"
										style="background: var(--bg-secondary); color: var(--text-secondary);">Cancel</button>
								</div>
							</div>
							<div class="flex flex-wrap gap-2">
								{#each page.columns as col, i}
									{@const k = kindOf(page.types?.[i] || '')}
									<div class="flex flex-col gap-1">
										<span class="text-[10px] font-[JetBrains_Mono,monospace]" style="color: var(--text-tertiary);">{col} <span style="color: var(--text-muted);">{page.types?.[i] || ''}</span></span>
										{#if k === 'bool'}
											<select bind:value={newRow[col]} class="w-40 px-2 py-1.5 rounded text-[12px] outline-none cursor-pointer" style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);">
												<option value="">(default)</option>
												<option value="true">true</option>
												<option value="false">false</option>
											</select>
										{:else if k === 'date'}
											<input type="date" bind:value={newRow[col]} class="w-44 px-2 h-8 rounded text-[12px] font-[JetBrains_Mono,monospace] outline-none" style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
										{:else if k === 'datetime'}
											<input type="datetime-local" step="1" bind:value={newRow[col]} class="w-52 px-2 h-8 rounded text-[12px] font-[JetBrains_Mono,monospace] outline-none" style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
										{:else if k === 'number'}
											<input type="text" inputmode="decimal" bind:value={newRow[col]} placeholder="123" spellcheck="false" autocomplete="off" class="w-40 px-2 h-8 rounded text-[12px] font-[JetBrains_Mono,monospace] outline-none" style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
										{:else}
											<input bind:value={newRow[col]} spellcheck="false" autocomplete="off" class="w-40 px-2 h-8 rounded text-[12px] font-[JetBrains_Mono,monospace] outline-none" style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border);" />
										{/if}
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<div class="flex-1 min-h-0 overflow-auto">
						{#if page && page.columns?.length}
							<table class="border-collapse" style="min-width: 100%;">
								<thead style="position: sticky; top: 0; z-index: 3;">
									<tr>
										<th class="px-2.5 py-2 text-center" style="width: 34px; position: sticky; left: 0; z-index: 4; background: var(--bg-tertiary); border-bottom: 1px solid var(--border); border-right: 1px solid var(--border);">
											<input type="checkbox" class="cbx" checked={allSelected} onchange={toggleAll} title="Select all on page" />
										</th>
										{#each page.columns as col, i}
											<th class="text-left px-3 py-2 text-[11px] font-semibold whitespace-nowrap" style="color: var(--text-secondary); background: var(--bg-tertiary); border-bottom: 1px solid var(--border);">
												{col}
												<span class="ml-1 text-[9.5px] font-normal font-[JetBrains_Mono,monospace]" style="color: var(--text-muted);">{page.types?.[i] || ''}</span>
											</th>
										{/each}
									</tr>
								</thead>
								<tbody>
									{#each page.rows as row, ri (ri)}
										{@const ctid = page.ctids?.[ri] || ''}
										{@const sel = isSel(ctid)}
										{@const rowBg = sel ? 'var(--active-bg)' : ri % 2 === 1 ? 'var(--bg-tertiary)' : 'var(--bg-secondary)'}
										<tr style="background: {rowBg};">
											<td class="px-2.5 py-1.5 text-center" style="position: sticky; left: 0; z-index: 1; background: {rowBg}; border-right: 1px solid var(--border);">
												<input type="checkbox" class="cbx" checked={sel} onchange={() => toggleRow(ctid)} disabled={!ctid} />
											</td>
											{#each row as cell, ci (ci)}
												{@const jv = jsonCell(cell, page.types?.[ci])}
												<td class="px-3 py-1.5 text-[12px] font-[JetBrains_Mono,monospace] whitespace-nowrap cell" style="color: var(--text-primary);"
													ondblclick={() => openEditCell(ri, ci)} title="Double-click to edit">
													{#if cell === null}
														<span class="italic" style="color: var(--text-muted);">NULL</span>
													{:else if jv !== undefined}
														<span class="inline-flex items-center gap-1.5 align-middle">
															<button onclick={(e) => { e.stopPropagation(); openJson(page.columns[ci], jv); }} title="View JSON"
																class="shrink-0 grid place-items-center w-5 h-5 rounded cursor-pointer" style="background: var(--bg-tertiary); color: var(--accent);">
																<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3H7a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2 2 2 0 0 1 2 2v5c0 1.1.9 2 2 2h1"/><path d="M16 3h1a2 2 0 0 1 2 2v5a2 2 0 0 0 2 2 2 2 0 0 0-2 2v5a2 2 0 0 1-2 2h-1"/></svg>
															</button>
															<span class="inline-block max-w-[320px] truncate align-bottom" style="color: var(--text-tertiary);">{cell}</span>
														</span>
													{:else}
														<span class="inline-block max-w-[420px] truncate align-bottom">{cell}</span>
													{/if}
												</td>
											{/each}
										</tr>
									{/each}
								</tbody>
							</table>
							{#if page.rows.length === 0}
								<div class="px-4 py-6 text-[12.5px]" style="color: var(--text-muted);">No rows.</div>
							{/if}
						{/if}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Delete confirm modal -->
{#if confirmDel}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center" style="background: rgba(0,0,0,0.5);" onclick={() => (confirmDel = null)}>
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="rounded-lg p-6 space-y-4 w-[380px] shadow-xl" style="background: var(--bg-secondary); border: 1px solid var(--border);" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-[14px] font-semibold" style="color: var(--text-primary);">Delete {confirmDel.ctids.length} row{confirmDel.ctids.length === 1 ? '' : 's'}</h3>
			<p class="text-[13px]" style="color: var(--text-tertiary);">
				Permanently delete {confirmDel.ctids.length} selected row{confirmDel.ctids.length === 1 ? '' : 's'} from <span class="font-[JetBrains_Mono,monospace] text-[12px]" style="color: var(--text-secondary);">{selectedTable?.schema}.{selectedTable?.name}</span>? This cannot be undone.
			</p>
			<div class="flex justify-end gap-2 pt-1">
				<button onclick={() => (confirmDel = null)} class="px-3 py-1.5 rounded-md text-[13px] font-medium cursor-pointer" style="background: var(--bg-tertiary); color: var(--text-secondary);">Cancel</button>
				<button onclick={doDelete} disabled={deleting} class="px-3 py-1.5 rounded-md text-[13px] font-medium cursor-pointer disabled:opacity-50 bg-red-500/10 text-red-500 hover:bg-red-500/20">{deleting ? 'Deleting…' : 'Delete'}</button>
			</div>
		</div>
	</div>
{/if}

<!-- Edit cell modal (the popup confirms before applying) -->
{#if editCell}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center" style="background: rgba(0,0,0,0.5);" onclick={() => (editCell = null)}>
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="rounded-lg p-5 space-y-3 w-[460px] shadow-xl" style="background: var(--bg-secondary); border: 1px solid var(--border);" onclick={(e) => e.stopPropagation()}>
			<div class="flex items-center gap-2">
				<h3 class="text-[14px] font-semibold" style="color: var(--text-primary);">Edit {editCell.column}</h3>
				<span class="text-[10.5px] font-[JetBrains_Mono,monospace]" style="color: var(--text-muted);">{editCell.type}</span>
			</div>
			<div class="text-[11px] font-[JetBrains_Mono,monospace]" style="color: var(--text-muted);">
				{selectedTable?.schema}.{selectedTable?.name} · ctid {editCell.ctid}
			</div>
			<div>
				<div class="text-[10.5px] uppercase tracking-wider mb-1" style="color: var(--text-tertiary);">Value</div>
				{#if editKind === 'bool'}
					<label class="flex items-center gap-2.5 px-3 py-2.5 rounded-md w-fit cursor-pointer" style="background: var(--input-bg); border: 1px solid var(--border); opacity: {editNull ? 0.5 : 1};">
						<input type="checkbox" class="cbx" checked={editValue === 'true'} disabled={editNull}
							onchange={(e) => (editValue = e.currentTarget.checked ? 'true' : 'false')} />
						<span class="text-[13px] font-[JetBrains_Mono,monospace]" style="color: var(--text-primary);">{editValue === 'true' ? 'true' : 'false'}</span>
					</label>
				{:else if editKind === 'number'}
					<input type="text" inputmode="decimal" bind:value={editValue} disabled={editNull} spellcheck="false"
						class="w-full px-2.5 h-9 rounded-md text-[13px] font-[JetBrains_Mono,monospace] outline-none"
						style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border); opacity: {editNull ? 0.5 : 1};" />
				{:else}
					<textarea bind:value={editValue} disabled={editNull} rows="3" spellcheck="false"
						class="w-full px-2.5 py-2 rounded-md text-[12.5px] font-[JetBrains_Mono,monospace] outline-none resize-y"
						style="background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--border); opacity: {editNull ? 0.5 : 1};"></textarea>
				{/if}
			</div>
			<label class="flex items-center gap-2 text-[12px] cursor-pointer w-fit" style="color: var(--text-secondary);">
				<input type="checkbox" class="cbx" bind:checked={editNull} />
				Set NULL
			</label>
			{#if error}
				<div class="rounded-md px-3 py-2 text-[11.5px] font-[JetBrains_Mono,monospace]" style="background: rgba(239,68,68,0.1); color: rgb(239,68,68);">{error}</div>
			{/if}
			<div class="flex justify-end gap-2 pt-1">
				<button onclick={() => (editCell = null)} class="px-3 py-1.5 rounded-md text-[13px] font-medium cursor-pointer" style="background: var(--bg-tertiary); color: var(--text-secondary);">Cancel</button>
				<button onclick={doUpdate} disabled={updating} class="px-3 py-1.5 rounded-md text-[13px] font-medium cursor-pointer disabled:opacity-50" style="background: var(--accent); color: var(--accent-text);">{updating ? 'Saving…' : 'Save change'}</button>
			</div>
		</div>
	</div>
{/if}

<!-- JSON viewer -->
{#if jsonView}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center p-6" style="background: rgba(0,0,0,0.5);" onclick={() => (jsonView = null)}>
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="rounded-lg w-[640px] max-w-full max-h-[80vh] flex flex-col shadow-xl" style="background: var(--bg-secondary); border: 1px solid var(--border);" onclick={(e) => e.stopPropagation()}>
			<div class="flex items-center justify-between px-4 py-3 shrink-0" style="border-bottom: 1px solid var(--border);">
				<div class="flex items-center gap-2 min-w-0">
					<span class="text-[13px] font-semibold font-[JetBrains_Mono,monospace] truncate" style="color: var(--text-primary);">{jsonView.column}</span>
					<span class="text-[10.5px] uppercase tracking-wider" style="color: var(--text-muted);">JSON</span>
				</div>
				<div class="flex items-center gap-2 shrink-0">
					<button onclick={copyJson} class="px-2.5 h-7 rounded text-[12px] font-medium cursor-pointer" style="background: var(--bg-tertiary); color: var(--text-secondary);">Copy</button>
					<button onclick={() => (jsonView = null)} class="px-2.5 h-7 rounded text-[12px] font-medium cursor-pointer" style="background: var(--bg-tertiary); color: var(--text-secondary);">Close</button>
				</div>
			</div>
			<pre class="flex-1 overflow-auto m-0 px-4 py-3 text-[12px] leading-[1.5] font-[JetBrains_Mono,monospace]" style="color: var(--text-primary);">{@html hlJson(jsonView.obj)}</pre>
		</div>
	</div>
{/if}

<style>
	input[type='checkbox'].cbx {
		appearance: none;
		-webkit-appearance: none;
		width: 15px;
		height: 15px;
		border: 1.5px solid var(--border);
		border-radius: 4px;
		background: var(--bg-secondary);
		cursor: pointer;
		position: relative;
		margin: 0;
		flex-shrink: 0;
		transition: background 0.12s, border-color 0.12s;
	}
	input[type='checkbox'].cbx:hover {
		border-color: var(--text-muted);
	}
	input[type='checkbox'].cbx:checked {
		background: var(--accent);
		border-color: var(--accent);
	}
	input[type='checkbox'].cbx:checked::after {
		content: '';
		position: absolute;
		left: 4px;
		top: 1px;
		width: 4px;
		height: 8px;
		border: solid #fff;
		border-width: 0 2px 2px 0;
		transform: rotate(45deg);
	}
	input[type='checkbox'].cbx:disabled {
		opacity: 0.3;
		cursor: default;
	}
	td.cell {
		cursor: cell;
	}
	td.cell:hover {
		background: var(--hover-bg);
	}
	:global(.j-key) {
		color: #79c0ff;
	}
	:global(.j-str) {
		color: #a5d6ff;
	}
	:global(.j-num) {
		color: #f0883e;
	}
	:global(.j-bool) {
		color: #d2a8ff;
	}
	:global(.j-nul) {
		color: var(--text-muted);
	}
</style>
