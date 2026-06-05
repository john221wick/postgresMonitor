<script>
	import { onMount } from 'svelte';
	import { GetRemoteMode } from './lib/api.js';
	import Dashboard from './pages/Dashboard.svelte';
	import Settings from './pages/Settings.svelte';
	import Terminal from './pages/Terminal.svelte';
	import Monitor from './pages/Monitor.svelte';
	import Database from './pages/Database.svelte';
	import postgresLogo from './lib/assets/postgres.png';

	let page = $state('dashboard');
	let dark = $state(true);
	let remoteMode = $state(false);
	let collapsed = $state(false);
	let paletteOpen = $state(false);
	let query = $state('');
	let paletteInput = $state(null);

	const routes = {
		'': 'dashboard',
		dashboard: 'dashboard',
		monitor: 'monitor',
		database: 'database',
		terminal: 'terminal',
		settings: 'settings'
	};

	const titles = {
		dashboard: 'Dashboard',
		monitor: 'Monitor',
		database: 'Database',
		terminal: 'Terminal',
		settings: 'Settings'
	};

	const icons = {
		dashboard:
			'<rect width="7" height="9" x="3" y="3" rx="1"/><rect width="7" height="5" x="14" y="3" rx="1"/><rect width="7" height="9" x="14" y="12" rx="1"/><rect width="7" height="5" x="3" y="16" rx="1"/>',
		monitor: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>',
		database: '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/>',
		terminal: '<polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>',
		settings:
			'<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>',
		search: '<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>',
		panel: '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/>',
		sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>',
		moon: '<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z"/>',
		expand:
			'<path d="M8 3H5a2 2 0 0 0-2 2v3M21 8V5a2 2 0 0 0-2-2h-3M3 16v3a2 2 0 0 0 2 2h3M16 21h3a2 2 0 0 0 2-2v-3"/>',
		shrink:
			'<path d="M8 3v3a2 2 0 0 1-2 2H3M21 8h-3a2 2 0 0 1-2-2V3M3 16h3a2 2 0 0 1 2 2v3M16 21v-3a2 2 0 0 1 2-2h3"/>'
	};

	function handleHash() {
		const hash = window.location.hash.replace('#/', '').replace('#', '');
		page = routes[hash] || 'dashboard';
	}

	onMount(async () => {
		const saved = localStorage.getItem('theme');
		dark = saved ? saved === 'dark' : true;
		applyTheme();

		collapsed = localStorage.getItem('sidebarCollapsed') === '1';

		try {
			remoteMode = await GetRemoteMode();
		} catch {}

		handleHash();
		window.addEventListener('hashchange', handleHash);
		window.addEventListener('keydown', onKey);
		return () => {
			window.removeEventListener('hashchange', handleHash);
			window.removeEventListener('keydown', onKey);
		};
	});

	function onKey(e) {
		const meta = e.metaKey || e.ctrlKey;
		if (meta && e.key.toLowerCase() === 'b') {
			e.preventDefault();
			toggleSidebar();
		} else if (meta && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			openPalette();
		} else if (e.key === 'Escape' && paletteOpen) {
			paletteOpen = false;
		}
	}

	function onModeChange(mode) {
		remoteMode = mode;
	}

	function applyTheme() {
		document.documentElement.classList.toggle('dark', dark);
		localStorage.setItem('theme', dark ? 'dark' : 'light');
	}

	function toggleTheme() {
		dark = !dark;
		applyTheme();
	}

	function toggleSidebar() {
		collapsed = !collapsed;
		localStorage.setItem('sidebarCollapsed', collapsed ? '1' : '0');
	}

	function navigate(route) {
		window.location.hash = '#/' + route;
		paletteOpen = false;
		query = '';
	}

	async function openPalette() {
		paletteOpen = true;
		query = '';
		await Promise.resolve();
		paletteInput?.focus();
	}

	const sections = $derived(
		remoteMode
			? [
					{ label: null, items: [{ route: 'dashboard', label: 'Dashboard', icon: 'dashboard' }] },
					{
						label: 'Cluster',
						items: [
							{ route: 'monitor', label: 'Monitor', icon: 'monitor' },
							{ route: 'database', label: 'Database', icon: 'database' }
						]
					},
					{
						label: 'Tools',
						items: [{ route: 'terminal', label: 'Terminal', icon: 'terminal' }]
					}
				]
			: [
					{ label: null, items: [{ route: 'dashboard', label: 'Dashboard', icon: 'dashboard' }] },
					{
						label: 'System',
						items: [
							{ route: 'monitor', label: 'Monitor', icon: 'monitor' },
							{ route: 'database', label: 'Database', icon: 'database' }
						]
					}
				]
	);

	const allItems = $derived(sections.flatMap((s) => s.items));
	const filtered = $derived(
		query.trim()
			? allItems.filter((i) => i.label.toLowerCase().includes(query.trim().toLowerCase()))
			: allItems
	);
</script>

{#snippet icon(name, size = 16, stroke = 1.75)}
	<svg
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width={stroke}
		stroke-linecap="round"
		stroke-linejoin="round"
	>
		{@html icons[name]}
	</svg>
{/snippet}

<div class="flex h-screen overflow-hidden" style="background: var(--bg-primary);">
	<aside
		class="shrink-0 overflow-hidden transition-[width] duration-200 ease-out"
		style="width: {collapsed ? '0px' : 'var(--sidebar-width)'}; background: var(--bg-sidebar); border-right: 1px solid var(--border-subtle);"
	>
		<div class="flex flex-col h-full" style="width: var(--sidebar-width);">
			<div class="flex items-center justify-between pl-4 pr-2.5 h-[var(--topbar-height)]">
				<div class="flex items-center gap-2">
					<img src={postgresLogo} alt="Postgres Monitor" class="w-5 h-5 rounded-[5px] object-contain" />
					<span class="text-[13.5px] font-semibold tracking-tight" style="color: var(--text-primary);"
						>Postgres Monitor</span
					>
				</div>
				<button
					onclick={toggleSidebar}
					title="Collapse sidebar (⌘B)"
					class="grid place-items-center w-7 h-7 rounded-md cursor-pointer transition-colors"
					style="color: var(--text-tertiary);"
					onmouseenter={(e) => {
						e.currentTarget.style.background = 'var(--hover-bg)';
						e.currentTarget.style.color = 'var(--text-primary)';
					}}
					onmouseleave={(e) => {
						e.currentTarget.style.background = 'transparent';
						e.currentTarget.style.color = 'var(--text-tertiary)';
					}}
				>
					{@render icon('panel', 16)}
				</button>
			</div>

			<div class="px-2.5 pt-1 pb-2">
				<button
					onclick={openPalette}
					class="flex items-center gap-2.5 w-full px-2.5 py-2 rounded-lg text-[13px] font-medium cursor-pointer transition-colors"
					style="color: var(--text-secondary);"
					onmouseenter={(e) => (e.currentTarget.style.background = 'var(--hover-bg)')}
					onmouseleave={(e) => (e.currentTarget.style.background = 'transparent')}
				>
					{@render icon('search', 15)}
					<span class="flex-1 text-left">Search</span>
					<span
						class="text-[10.5px] font-mono px-1.5 py-0.5 rounded"
						style="background: var(--bg-tertiary); color: var(--text-tertiary);">⌘K</span
					>
				</button>
			</div>

			<nav class="flex-1 px-2.5 py-1 overflow-y-auto">
				{#each sections as section}
					{#if section.label}
						<div
							class="text-[10.5px] font-semibold uppercase tracking-[0.08em] px-2.5 pt-4 pb-1"
							style="color: var(--text-muted);"
						>
							{section.label}
						</div>
					{/if}
					{#each section.items as item (item.route)}
						{@const active = page === item.route}
						<button
							onclick={() => navigate(item.route)}
							class="flex items-center gap-2.5 w-full text-left px-2.5 py-1.5 rounded-lg text-[13px] font-medium transition-colors cursor-pointer mb-0.5"
							style="color: {active ? 'var(--text-primary)' : 'var(--text-tertiary)'}; background: {active
								? 'var(--active-bg)'
								: 'transparent'};"
							onmouseenter={(e) => {
								if (!active) {
									e.currentTarget.style.color = 'var(--text-primary)';
									e.currentTarget.style.background = 'var(--hover-bg)';
								}
							}}
							onmouseleave={(e) => {
								if (!active) {
									e.currentTarget.style.color = 'var(--text-tertiary)';
									e.currentTarget.style.background = 'transparent';
								}
							}}
						>
							<span style="color: {active ? 'var(--accent)' : 'inherit'};"
								>{@render icon(item.icon, 16)}</span
							>
							{item.label}
						</button>
					{/each}
				{/each}
			</nav>

			<div class="p-2.5" style="border-top: 1px solid var(--border-subtle);">
				<button
					onclick={() => navigate('settings')}
					class="flex items-center gap-2.5 w-full px-2 py-1.5 rounded-lg cursor-pointer transition-colors"
					style="background: {page === 'settings' ? 'var(--active-bg)' : 'transparent'};"
					onmouseenter={(e) => {
						if (page !== 'settings') e.currentTarget.style.background = 'var(--hover-bg)';
					}}
					onmouseleave={(e) => {
						if (page !== 'settings') e.currentTarget.style.background = 'transparent';
					}}
				>
					<div
						class="w-7 h-7 rounded-full grid place-items-center text-[12px] font-semibold shrink-0"
						style="background: var(--accent); color: var(--accent-text);"
					>
						L
					</div>
					<div class="flex-1 min-w-0 text-left">
						<div class="text-[12.5px] font-semibold truncate" style="color: var(--text-primary);">
							Local
						</div>
						<div class="text-[11px] truncate" style="color: var(--text-tertiary);">
							{remoteMode ? 'Remote · cluster' : 'Inplace · local'}
						</div>
					</div>
					<span style="color: var(--text-muted);">{@render icon('settings', 15)}</span>
				</button>
			</div>
		</div>
	</aside>

	<div class="flex-1 flex flex-col min-w-0">
		<header
			class="flex items-center gap-2 px-3 shrink-0 h-[var(--topbar-height)]"
			style="border-bottom: 1px solid var(--border-subtle); background: var(--bg-primary);"
		>
			{#if collapsed}
				<button
					onclick={toggleSidebar}
					title="Open sidebar (⌘B)"
					class="grid place-items-center w-7 h-7 rounded-md cursor-pointer transition-colors"
					style="color: var(--text-tertiary);"
					onmouseenter={(e) => {
						e.currentTarget.style.background = 'var(--hover-bg)';
						e.currentTarget.style.color = 'var(--text-primary)';
					}}
					onmouseleave={(e) => {
						e.currentTarget.style.background = 'transparent';
						e.currentTarget.style.color = 'var(--text-tertiary)';
					}}
				>
					{@render icon('panel', 16)}
				</button>
			{/if}
			<div class="flex items-center gap-1.5 text-[13px] font-medium" style="color: var(--text-secondary);">
				<span style="color: var(--text-tertiary);">postgres</span>
				<span style="color: var(--text-muted);">/</span>
				<span style="color: var(--text-primary);">{titles[page]}</span>
			</div>

			<div class="flex-1"></div>

			<button
				onclick={openPalette}
				title="Search (⌘K)"
				class="grid place-items-center w-7 h-7 rounded-md cursor-pointer transition-colors"
				style="color: var(--text-tertiary);"
				onmouseenter={(e) => {
					e.currentTarget.style.background = 'var(--hover-bg)';
					e.currentTarget.style.color = 'var(--text-primary)';
				}}
				onmouseleave={(e) => {
					e.currentTarget.style.background = 'transparent';
					e.currentTarget.style.color = 'var(--text-tertiary)';
				}}
			>
				{@render icon('search', 16)}
			</button>
			<button
				onclick={toggleTheme}
				title="Toggle theme"
				class="grid place-items-center w-7 h-7 rounded-md cursor-pointer transition-colors"
				style="color: var(--text-tertiary);"
				onmouseenter={(e) => {
					e.currentTarget.style.background = 'var(--hover-bg)';
					e.currentTarget.style.color = 'var(--text-primary)';
				}}
				onmouseleave={(e) => {
					e.currentTarget.style.background = 'transparent';
					e.currentTarget.style.color = 'var(--text-tertiary)';
				}}
			>
				{@render icon(dark ? 'sun' : 'moon', 16)}
			</button>
		</header>

		<main class="flex-1 overflow-y-auto" style="background: var(--bg-primary);">
			{#if page === 'dashboard'}
				<Dashboard {remoteMode} {onModeChange} />
			{:else if page === 'monitor'}
				<Monitor {remoteMode} />
			{:else if page === 'database'}
				<Database {remoteMode} />
			{:else if page === 'settings'}
				<Settings {dark} {toggleTheme} {remoteMode} />
			{/if}

			<div class="h-full" style="display: {page === 'terminal' ? 'block' : 'none'};">
				<Terminal active={page === 'terminal'} />
			</div>
		</main>
	</div>
</div>

{#if paletteOpen}
	<div
		class="fixed inset-0 z-50 flex items-start justify-center pt-[18vh] px-4"
		style="background: rgba(0,0,0,0.35);"
		onclick={() => (paletteOpen = false)}
		onkeydown={() => {}}
		role="presentation"
	>
		<div
			class="w-full max-w-[520px] rounded-xl overflow-hidden shadow-2xl"
			style="background: var(--bg-elevated); border: 1px solid var(--border);"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
			role="dialog"
			tabindex="-1"
		>
			<div
				class="flex items-center gap-2.5 px-3.5 h-11"
				style="border-bottom: 1px solid var(--border-subtle);"
			>
				<span style="color: var(--text-tertiary);">{@render icon('search', 16)}</span>
				<input
					bind:this={paletteInput}
					bind:value={query}
					onkeydown={(e) => {
						if (e.key === 'Enter' && filtered[0]) navigate(filtered[0].route);
					}}
					placeholder="Jump to a page…"
					class="flex-1 bg-transparent outline-none text-[14px]"
					style="color: var(--text-primary);"
				/>
				<span
					class="text-[10.5px] font-mono px-1.5 py-0.5 rounded"
					style="background: var(--bg-tertiary); color: var(--text-tertiary);">esc</span
				>
			</div>
			<div class="max-h-[320px] overflow-y-auto p-1.5">
				{#each filtered as item (item.route)}
					<button
						onclick={() => navigate(item.route)}
						class="flex items-center gap-3 w-full px-2.5 py-2 rounded-lg text-[13px] font-medium text-left cursor-pointer transition-colors"
						style="color: var(--text-secondary);"
						onmouseenter={(e) => {
							e.currentTarget.style.background = 'var(--hover-bg)';
							e.currentTarget.style.color = 'var(--text-primary)';
						}}
						onmouseleave={(e) => {
							e.currentTarget.style.background = 'transparent';
							e.currentTarget.style.color = 'var(--text-secondary)';
						}}
					>
						<span style="color: var(--text-tertiary);">{@render icon(item.icon, 16)}</span>
						{item.label}
					</button>
				{:else}
					<div class="px-3 py-6 text-center text-[13px]" style="color: var(--text-muted);">
						No matches
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}