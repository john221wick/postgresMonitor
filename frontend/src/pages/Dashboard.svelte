<script>
	import { onMount } from 'svelte';
	import { SetRemoteMode, GetLocalMonitor } from '../lib/api.js';
	import Nodes from './Nodes.svelte';

	let { remoteMode = false, onModeChange = () => {} } = $props();
	let localNode = $state(null);
	let switching = $state(false);
	let interval;

	async function refresh() {
		if (remoteMode) {
			localNode = null;
			return;
		}
		try {
			const nodes = (await GetLocalMonitor()) || [];
			localNode = nodes[0] || null;
		} catch (e) {
			console.error('Dashboard refresh failed:', e);
		}
	}

	async function toggleMode() {
		switching = true;
		try {
			const newMode = !remoteMode;
			await SetRemoteMode(newMode);
			onModeChange(newMode);
			localNode = null;
			await refresh();
		} catch (e) {
			console.error('Mode switch failed:', e);
		} finally {
			switching = false;
		}
	}

	onMount(() => {
		refresh();
		interval = setInterval(refresh, 5000);
		return () => clearInterval(interval);
	});

	function memPct(used, total) {
		if (!total) return 0;
		return Math.round((used / total) * 100);
	}
</script>

<div class="p-8 space-y-6 max-w-[1000px]">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-lg font-semibold" style="color: var(--text-primary);">Dashboard</h1>
			<p class="text-[13px] mt-0.5" style="color: var(--text-tertiary);">
				{remoteMode ? 'Cluster nodes' : 'System overview'}
			</p>
		</div>
		<div class="flex items-center gap-1 rounded-lg p-0.5" style="background: var(--bg-tertiary);">
			<button
				onclick={toggleMode}
				disabled={switching}
				class="px-3 py-1.5 rounded-md text-[12px] font-medium transition-colors cursor-pointer"
				style="background: {!remoteMode ? 'var(--accent)' : 'transparent'}; color: {!remoteMode ? 'var(--accent-text)' : 'var(--text-tertiary)'};"
			>
				Inplace
			</button>
			<button
				onclick={toggleMode}
				disabled={switching}
				class="px-3 py-1.5 rounded-md text-[12px] font-medium transition-colors cursor-pointer"
				style="background: {remoteMode ? 'var(--accent)' : 'transparent'}; color: {remoteMode ? 'var(--accent-text)' : 'var(--text-tertiary)'};"
			>
				Remote
			</button>
		</div>
	</div>

	{#if remoteMode}
		<Nodes embedded />
	{:else if localNode}
		<div class="grid grid-cols-4 gap-3">
			<div class="rounded-lg p-4" style="background: var(--bg-secondary); border: 1px solid var(--border);">
				<div class="text-[11px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">CPU</div>
				<div class="text-2xl font-semibold mt-2 font-[JetBrains_Mono,monospace]" style="color: var(--text-primary);">
					{(localNode.host?.cpuPercent || 0).toFixed(0)}%
				</div>
				<div class="text-[12px] mt-1" style="color: var(--text-secondary);">
					{localNode.host?.cpuCores || 0} cores
				</div>
			</div>

			<div class="rounded-lg p-4" style="background: var(--bg-secondary); border: 1px solid var(--border);">
				<div class="text-[11px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Memory</div>
				<div class="text-2xl font-semibold mt-2 font-[JetBrains_Mono,monospace]" style="color: var(--text-primary);">
					{memPct(localNode.host?.memUsedMB, localNode.host?.memTotalMB)}%
				</div>
				<div class="text-[12px] mt-1" style="color: var(--text-secondary);">
					{((localNode.host?.memUsedMB || 0) / 1024).toFixed(1)} / {((localNode.host?.memTotalMB || 0) / 1024).toFixed(1)} GB
				</div>
			</div>

			<div class="rounded-lg p-4" style="background: var(--bg-secondary); border: 1px solid var(--border);">
				<div class="text-[11px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Containers</div>
				<div class="text-2xl font-semibold mt-2 font-[JetBrains_Mono,monospace]" style="color: var(--text-primary);">
					{localNode.containers?.containers?.length || 0}
				</div>
				<div class="text-[12px] mt-1" style="color: var(--text-secondary);">
					{localNode.containers?.available ? 'Docker' : 'unavailable'}
				</div>
			</div>

			<div class="rounded-lg p-4" style="background: var(--bg-secondary); border: 1px solid var(--border);">
				<div class="text-[11px] font-medium uppercase tracking-wider" style="color: var(--text-tertiary);">Host</div>
				<div class="text-[15px] font-semibold mt-2 truncate" style="color: var(--text-primary);">
					{localNode.nodeName}
				</div>
				<div class="text-[12px] mt-1 truncate" style="color: var(--text-secondary);">
					{localNode.host?.osName || '—'}
				</div>
			</div>
		</div>
		<p class="text-[12px]" style="color: var(--text-muted);">
			Open <strong style="color: var(--text-secondary);">Monitor</strong> for process lists and container details.
		</p>
	{:else}
		<div class="flex items-center justify-center h-32" style="color: var(--text-muted);">Loading...</div>
	{/if}
</div>