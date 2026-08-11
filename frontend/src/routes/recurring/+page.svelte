<script lang="ts">
	import { onMount } from 'svelte';
	import { recurring, categories, accounts } from '$lib/api';
	import Icon from '$lib/icons.svelte';
	import type { RecurringTransaction, Category, Account, Transaction } from '$lib/types';

	let list = $state<RecurringTransaction[]>([]);
	let catList = $state<Category[]>([]);
	let accList = $state<Account[]>([]);
	let catMap = $state<Map<string,Category>>(new Map());
	let accMap = $state<Map<string,Account>>(new Map());

	// Expanded row for history
	let expandedId = $state('');
	let history = $state<Transaction[]>([]);
	let loadingHistory = $state(false);

	// Create/Edit modal
	let showModal = $state(false);
	let editId = $state('');
	let selAccount = $state('');
	let selCategory = $state('');
	let amount = $state('');
	let description = $state('');
	let intervalKind = $state('monthly');
	let intervalN = $state(1);
	let nextDue = $state(new Date().toISOString().slice(0,10));

	const intervals: Record<string,string> = { monthly:'monatlich', quarterly:'quartalsweise', yearly:'jährlich' };

	onMount(async () => {
		const [a,c] = await Promise.all([accounts.list(), categories.list()]);
		accList = a; catList = c;
		catMap = new Map(c.map(x => [x.id, x]));
		accMap = new Map(a.map(x => [x.id, x]));
		if (a.length > 0) selAccount = a[0].id;
		await load();
	});
	async function load() { list = await recurring.list(); }

	async function saveR() {
		try {
			const data = {
				account_id: selAccount, category_id: selCategory,
				amount: parseFloat(amount), description,
				interval_kind: intervalKind, interval_n: intervalN, next_due: nextDue
			};
			if (editId) {
				await recurring.update(editId, data);
			} else {
				await recurring.create(data);
			}
			resetForm();
			await load();
		} catch(e: any) { alert('Fehler: ' + e.message); }
	}

	function editR(rt: RecurringTransaction) {
		editId = rt.id;
		selAccount = rt.account_id;
		selCategory = rt.category_id;
		amount = String(rt.amount);
		description = rt.description;
		intervalKind = rt.interval_kind;
		intervalN = rt.interval_n;
		nextDue = rt.next_due;
		showModal = true;
	}

	function resetForm() {
		editId = ''; amount = ''; description = ''; showModal = false;
		intervalKind = 'monthly'; intervalN = 1;
		nextDue = new Date().toISOString().slice(0,10);
	}

	async function toggle(id: string) { await recurring.toggle(id); await load(); }
	async function delR(id: string) {
		if (!confirm('Löschen?')) return;
		await recurring.del(id); await load();
	}
	async function toggleHistory(id: string) {
		if (expandedId === id) { expandedId = ''; history = []; return; }
		expandedId = id;
		loadingHistory = true;
		try { history = await recurring.history(id) || []; }
		catch(e) { history = []; }
		loadingHistory = false;
	}
	async function processAll() {
		const r = await recurring.process();
		alert(`${r.processed} Daueraufträge verarbeitet`);
		await load();
	}
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') resetForm(); }} />

<h1 class="page-title">Daueraufträge</h1>

<div class="row" style="margin-bottom:1rem">
	<button class="primary" onclick={() => { editId = ''; showModal = true; }}><Icon name="plus" /> Neuer Dauerauftrag</button>
	<button onclick={processAll} style="border-color:var(--gold);color:var(--gold)">⚡ Jetzt ausführen</button>
</div>

<div class="card">
	<table>
		<thead><tr><th>Beschreibung</th><th>Konto</th><th>Kategorie</th><th>Betrag</th><th>Intervall</th><th>Nächster</th><th></th><th></th><th></th></tr></thead>
		<tbody>
			{#each list as rt}
				{@const cat = catMap.get(rt.category_id)}
				{@const acc = accMap.get(rt.account_id)}
				<tr onclick={() => toggleHistory(rt.id)} style="cursor:pointer">
					<td>{rt.description || '—'}</td>
					<td style="color:var(--muted);font-size:.8rem">{acc?.name}</td>
					<td>
						{#if cat}<span class="badge" style="background:{cat.color}18;color:{cat.color}">{cat.name}</span>{/if}
					</td>
					<td class="amount" class:income={rt.amount >= 0} class:expense={rt.amount < 0}>
						{rt.amount >= 0 ? '+' : ''}{rt.amount.toFixed(2)} €
					</td>
					<td style="color:var(--muted);font-size:.82rem">
						{intervals[rt.interval_kind]}{rt.interval_n > 1 ? ` (×${rt.interval_n})` : ''}
					</td>
					<td style="color:var(--muted);font-size:.82rem">{rt.next_due}</td>
					<td>
						<button onclick={() => toggle(rt.id)} class:primary={rt.active} class="small" style="min-width:70px">
							{rt.active ? 'Aktiv' : 'Pausiert'}
						</button>
					</td>
					<td><button class="ghost small" onclick={() => editR(rt)} title="Bearbeiten">✎</button></td>
					<td><button class="ghost small" onclick={(e) => { e.stopPropagation(); delR(rt.id); }}><Icon name="trash" /></button></td>
				</tr>
				{#if expandedId === rt.id}
					<tr><td colspan="9" style="background:var(--surface2);padding:0">
						{#if loadingHistory}
							<div style="padding:1rem;color:var(--muted)">Lade Verlauf…</div>
						{:else if history.length === 0}
							<div style="padding:1rem;color:var(--muted)">Keine vergangenen Buchungen gefunden.</div>
						{:else}
							<div style="padding:.5rem 1rem;font-size:.82rem">
								{#each history as hx}
									<div style="padding:.3rem 0;border-bottom:1px solid var(--border);display:flex;justify-content:space-between">
										<span>{hx.date} — {hx.description || '—'}</span>
										<span class="amount" class:income={hx.amount >= 0} class:expense={hx.amount < 0}>{hx.amount >= 0 ? '+' : ''}{hx.amount.toFixed(2)} €</span>
									</div>
								{/each}
							</div>
						{/if}
					</td></tr>
				{/if}
			{/each}
			{#if list.length === 0}
				<tr><td colspan="9" style="color:var(--muted);text-align:center;padding:2.5rem">Keine Daueraufträge</td></tr>
			{/if}
		</tbody>
	</table>
</div>

{#if showModal}
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={resetForm} role="presentation">
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>{editId ? 'Dauerauftrag bearbeiten' : 'Neuer Dauerauftrag'}</h2>
		<div class="form-group">
			<label for="r-desc">Beschreibung</label>
			<input id="r-desc" bind:value={description} placeholder="Miete" />
		</div>
		<div class="form-group">
			<label for="r-acct">Konto</label>
			<select id="r-acct" bind:value={selAccount}>
				{#each accList as a}<option value={a.id}>{a.name}</option>{/each}
			</select>
		</div>
		<div class="form-group">
			<label for="r-cat">Kategorie</label>
			<select id="r-cat" bind:value={selCategory}>
				<option value="">—</option>
				{#each catList as c}<option value={c.id}>{c.name}</option>{/each}
			</select>
		</div>
		<div class="form-group">
			<label for="r-amt">Betrag (€) — negativ für Ausgaben</label>
			<input id="r-amt" type="number" step="0.01" bind:value={amount} placeholder="-800" />
		</div>
		<div class="row">
			<div class="form-group" style="flex:2">
				<label for="r-iv">Intervall</label>
				<select id="r-iv" bind:value={intervalKind}>
					<option value="monthly">Monatlich</option>
					<option value="quarterly">Quartalsweise</option>
					<option value="yearly">Jährlich</option>
				</select>
			</div>
			<div class="form-group" style="flex:1">
				<label for="r-ivn">Alle N</label>
				<input id="r-ivn" type="number" min="1" bind:value={intervalN} />
			</div>
		</div>
		<div class="modal-actions">
			<button onclick={resetForm}>Abbrechen</button>
			<button class="primary" onclick={saveR}>{editId ? 'Speichern' : 'Erstellen'}</button>
		</div>
	</div>
</div>
{/if}
