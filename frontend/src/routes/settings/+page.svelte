<script lang="ts">
	import { onMount } from 'svelte';
	import { accounts, categories as catApi } from '$lib/api';
	import Icon from '$lib/icons.svelte';
	import type { Account, Category } from '$lib/types';

	let accList: Account[] = $state([]);
	let catList: Category[] = $state([]);

	// Account form
	let showAccModal = $state(false);
	let accName = $state('');
	let accType = $state('checking');

	// Category form
	let showCatModal = $state(false);
	let editCatId = $state('');
	let catName = $state('');
	let catColor = $state('#D4A574');
	let catIcon = $state('');
	let catIncome = $state(false);

	onMount(async () => {
		try {
			const [a, c] = await Promise.all([accounts.list(), catApi.list()]);
			accList = a;
			catList = c;
		} catch(e) { console.error(e); }
	});

	// Accounts
	async function createAcc() {
		if (!accName) return;
		await accounts.create({ name: accName, type: accType });
		accName = ''; showAccModal = false;
		const a = await accounts.list();
		accList = a;
	}
	async function delAcc(id: string) {
		if (!confirm('Konto löschen?')) return;
		await accounts.del(id);
		accList = await accounts.list();
	}

	// Categories
	async function saveCat() {
		if (!catName) return;
		if (editCatId) {
			await catApi.update(editCatId, { name: catName, color: catColor });
		} else {
			await catApi.create({ name: catName, color: catColor, icon: catIcon, is_income: catIncome });
		}
		resetCat();
		catList = await catApi.list();
	}
	function editCat(c: Category) {
		editCatId = c.id;
		catName = c.name;
		catColor = c.color;
		catIncome = c.is_income;
		catIcon = '';
		showCatModal = true;
	}
	function resetCat() {
		editCatId = ''; catName = ''; catIcon = ''; showCatModal = false;
		catColor = '#D4A574'; catIncome = false;
	}
	async function delCat(id: string) {
		if (!confirm('Kategorie löschen?')) return;
		await catApi.del(id);
		catList = await catApi.list();
	}
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { showAccModal = false; showCatModal = false; } }} />

<h1 class="page-title">Einstellungen</h1>

<!-- Konten -->
<section style="margin-bottom:2rem">
	<div class="row" style="margin-bottom:.75rem">
		<h2 style="font-size:.9rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em;flex:1">Konten</h2>
		<button class="primary small" onclick={() => showAccModal = true}>
			<Icon name="plus" /> Konto
		</button>
	</div>
	<div class="card">
		{#if !accList || accList.length === 0}
			<p style="color:var(--muted);padding:.5rem 0">Keine Konten angelegt.</p>
		{:else}
			{#each accList as a (a.id)}
				<div class="row" style="padding:.4rem 0;border-bottom:1px solid var(--border);justify-content:space-between">
					<div style="display:flex;align-items:center;gap:.5rem">
						<span style="font-size:.85rem;font-weight:600">{a.name}</span>
						<span style="font-size:.72rem;color:var(--muted)">{a.type}</span>
					</div>
					<div class="row" style="gap:.5rem">
						<span class="amount" class:positive={a.balance >= 0} class:negative={a.balance < 0}>
							{a.balance.toFixed(2)} {a.currency}
						</span>
						<button class="ghost small" onclick={() => delAcc(a.id)}><Icon name="trash" /></button>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</section>

<!-- Kategorien -->
<section>
	<div class="row" style="margin-bottom:.75rem">
		<h2 style="font-size:.9rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em;flex:1">Kategorien</h2>
		<button class="primary small" onclick={() => { resetCat(); showCatModal = true; }}>
			<Icon name="plus" /> Kategorie
		</button>
	</div>
	<div class="card">
		{#if !catList || catList.length === 0}
			<p style="color:var(--muted);padding:.5rem 0">Keine Kategorien.</p>
		{:else}
			{#each catList as c (c.id)}
				<div class="row" style="padding:.4rem 0;border-bottom:1px solid var(--border);justify-content:space-between">
					<div style="display:flex;align-items:center;gap:.55rem">
						<span style="background:{c.color};width:24px;height:24px;border-radius:4px;display:flex;align-items:center;justify-content:center;font-size:.8rem">
							<span style="filter:grayscale(1) brightness(10)">{c.icon || '—'}</span>
						</span>
						<span style="font-size:.85rem">{c.name}</span>
						<span class="badge" style="font-size:.68rem;background:var(--surface2);color:var(--muted);font-weight:400">
							{c.is_income ? 'Einnahme' : 'Ausgabe'}
						</span>
					</div>
					<div style="display:flex;gap:.3rem">
						<button class="ghost small" onclick={() => editCat(c)} title="Bearbeiten">✎</button>
						<button class="ghost small" onclick={() => delCat(c.id)}><Icon name="trash" /></button>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</section>

<!-- Modal: Account -->
{#if showAccModal}
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={() => showAccModal = false} role="presentation">
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>Neues Konto</h2>
		<div class="form-group">
			<label for="acc-name">Name</label>
			<input id="acc-name" bind:value={accName} placeholder="Girokonto" />
		</div>
		<div class="form-group">
			<label for="acc-type">Typ</label>
			<select id="acc-type" bind:value={accType}>
				<option value="checking">Girokonto</option>
				<option value="savings">Sparkonto</option>
				<option value="investment">Depot</option>
			</select>
		</div>
		<div class="modal-actions">
			<button onclick={() => showAccModal = false}>Abbrechen</button>
			<button class="primary" onclick={createAcc}>Erstellen</button>
		</div>
	</div>
</div>
{/if}

<!-- Modal: Category -->
{#if showCatModal}
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={() => showCatModal = false} role="presentation">
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>{editCatId ? 'Kategorie bearbeiten' : 'Neue Kategorie'}</h2>
		<div class="form-group">
			<label for="cat-name">Name</label>
			<input id="cat-name" bind:value={catName} placeholder="z.B. Lebensmittel" />
		</div>
		<div class="form-group">
			<label for="cat-icon">Icon</label>
			<input id="cat-icon" bind:value={catIcon} placeholder="🛒" style="width:80px" />
		</div>
		<div class="form-group">
			<label for="cat-color">Farbe</label>
			<input id="cat-color" type="color" bind:value={catColor} style="width:48px;height:32px;padding:2px" />
		</div>
		<div class="row" style="margin-bottom:.85rem">
			<button class:primary={!catIncome} onclick={() => catIncome = !catIncome} style="font-size:.82rem">
				{catIncome ? 'Einnahme' : 'Ausgabe'}
			</button>
		</div>
		<div class="modal-actions">
			<button onclick={() => showCatModal = false}>Abbrechen</button>
			<button class="primary" onclick={saveCat}>{editCatId ? 'Speichern' : 'Erstellen'}</button>
		</div>
	</div>
</div>
{/if}
