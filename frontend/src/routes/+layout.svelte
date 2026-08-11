<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import Icon from '$lib/icons.svelte';

	let { children } = $props();

	const tabs = [
		{ href: '/', label: 'Dashboard', icon: 'chart' },
		{ href: '/transactions', label: 'Transaktionen', icon: 'list' },
		{ href: '/recurring', label: 'Daueraufträge', icon: 'repeat' },
		{ href: '/settings', label: 'Einstellungen', icon: 'settings' },
	];

	function active(path: string) {
		if (path === '/') return $page.url.pathname === '/';
		return $page.url.pathname.startsWith(path);
	}
</script>

<div class="layout">
	<header class="topbar">
		<span class="topbar-logo">Fulla</span>
	</header>
	<div class="body">
		<nav class="sidebar">
			{#each tabs as t}
				<a href={t.href} class:active={active(t.href)}>
					<Icon name={t.icon} /> {t.label}
				</a>
			{/each}
		</nav>
		<main class="main">
			{@render children()}
		</main>
	</div>
</div>
