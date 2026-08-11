<script lang="ts">
	import { onMount } from 'svelte';
	import { transactions as txApi, categories, accounts, importCsv, recurring } from '$lib/api';
	import Icon from '$lib/icons.svelte';
	import type { Transaction, Category, Account } from '$lib/types';

	let txList = $state<Transaction[]>([]);
	let catList = $state<Category[]>([]);
	let accList = $state<Account[]>([]);
	let catMap = $state<Map<string,Category>>(new Map());

	let selectedAccount = $state('');
	let statusFilter = $state(''); // '' = all, 'pending', 'posted'
	let monthFilter = $state(''); // '' = all, '2026-08' etc.
	let searchFilter = $state('');
	let selected = $state<Set<string>>(new Set());

	let showModal = $state(false);
	let selectedCategory = $state('');
	let amount = $state('');
	let description = $state('');
	let date = $state(new Date().toISOString().slice(0,10));
	let isExpense = $state(true);

	// Bulk edit
	let bulkCat = $state('');

	onMount(async () => {
		try {
			[accList, catList] = await Promise.all([accounts.list(), categories.list()]);
			catMap = new Map(catList.map(c => [c.id, c]));
			if ((accList||[]).length > 0) selectedAccount = accList[0].id;
		} catch(e) { console.error(e); }
	});

	async function loadTx() {
		try {
			const result = await txApi.list({ account: selectedAccount || undefined, status: statusFilter || undefined, month: monthFilter || undefined, search: searchFilter || undefined });
			if (Array.isArray(result)) {
				txList = result;
				selected = new Set();
			}
		} catch(e) {
			console.error('loadTx failed:', e);
		}
	}
	$effect(() => { selectedAccount; statusFilter; monthFilter; searchFilter; loadTx(); });
	function toggleSelect(id: string) {
		const next = new Set(selected);
		if (next.has(id)) next.delete(id); else next.add(id);
		selected = next;
	}
	function toggleAll() {
		if (selected.size === (txList||[]).length) { selected = new Set(); return; }
		selected = new Set((txList||[]).filter(t => t.status === 'pending').map(t => t.id));
	}

	// Per-row quick edit for pending
	let editId = $state('');
	let quickCat = $state('');

	async function quickPost(id: string) {
		if (!quickCat) return;
		try {
			await txApi.bulkUpdate({ ids: [id], category_id: quickCat, status: 'posted' });
			editId = ''; quickCat = '';
			await loadTx();
		} catch(e) { alert('Fehler: ' + e); }
	}

	async function bulkPost() {
		const ids = [...selected].filter(id => {
			const t = txList.find(x => x.id === id);
			return t?.status === 'pending';
		});
		if (ids.length === 0) return;
		try {
			await txApi.bulkUpdate({ ids, category_id: bulkCat || undefined, status: 'posted' });
			bulkCat = '';
			selected = new Set();
			await loadTx();
		} catch(e) { alert('Fehler: ' + e); }
	}

	async function createTx() {
		const cat = catList.find(c => c.id === selectedCategory);
		const amt = parseFloat(amount) * (isExpense && cat && !cat.is_income ? -1 : 1);
		await txApi.create({ account_id: selectedAccount, category_id: selectedCategory, amount: amt, description, date });
		showModal = false; amount = ''; description = ''; date = new Date().toISOString().slice(0,10);
		await loadTx();
	}

	async function toggleExclude(id: string) {
		await txApi.toggleExclude(id); await loadTx();
	}
	async function delTx(id: string) {
		if (!confirm('Transaktion löschen?')) return;
		await txApi.del(id);
		await loadTx();
	}
	function openEditTx(tx: Transaction) {
		editTxId = tx.id; editTxAmount = String(tx.amount);
		editTxDesc = tx.description; editTxCat = tx.category_id;
		editTxDate = tx.date; showEditTx = true;
	}
	async function saveEditTx() {
		const amt = parseFloat(editTxAmount);
		if (isNaN(amt)) return;
		await txApi.update(editTxId, {
			account_id: selectedAccount, category_id: editTxCat,
			amount: amt, description: editTxDesc, date: editTxDate,
		});
		showEditTx = false; await loadTx();
	}

	// Recurring creation form
	let showRecurringModal = $state(false);
	let recTx = $state<Transaction | null>(null);
	let recInterval = $state('monthly');
	let recIntervalN = $state(1);

	// Edit modal for transactions
	let showEditTx = $state(false);
	let editTxId = $state('');
	let editTxAmount = $state('');
	let editTxDesc = $state('');
	let editTxCat = $state('');
	let editTxDate = $state('');

	async function handleCsv(e: Event) {
		const f = (e.target as HTMLInputElement).files?.[0];
		if (!f || !selectedAccount) return;
		try {
			const r = await importCsv.upload(f, selectedAccount);
			alert(`Importiert: ${r.imported}, Automatisch: ${r.auto_posted || 0}, Übersprungen: ${r.skipped}`);
			if (r.imported > 0) statusFilter = 'pending';
			await loadTx();
		} catch (err: any) { alert('Fehler: ' + err.message); }
	}

	async function createRecurringFromTx(tx: Transaction) {
		recTx = tx;
		recInterval = 'monthly';
		recIntervalN = 1;
		showRecurringModal = true;
	}

	async function saveRecurring() {
		console.log('saveRecurring called, recTx:', recTx);
		if (!recTx) { alert('Keine Transaktion ausgewählt.'); return; }
		try {
			console.log('sending recurring.create...');
			const result: any = await recurring.create({
				account_id: recTx.account_id,
				category_id: recTx.category_id || '',
				amount: recTx.amount,
				description: recTx.description,
				interval_kind: recInterval,
				interval_n: recIntervalN,
			});
			console.log('result:', result);
			const retro = result?.retro_matched || 0;
			alert(`Dauerauftrag angelegt!${retro > 0 ? ` ${retro} frühere Buchungen automatisch zugeordnet.` : ''}`);
			showRecurringModal = false;
			await loadTx();
		} catch(e: any) { console.error('saveRecurring error:', e); alert('Fehler: ' + e.message); }
	}

	function closeModal() { showModal = false; }
	function fmtAmt(a: number) { return (a >= 0 ? '+' : '') + a.toFixed(2) + ' €'; }
	function catById(id: string) { return catMap.get(id); }

	const pendingCount = $derived((txList || []).filter(t => t.status === 'pending').length);
	const months = $derived.by(() => {
		const now = new Date(); const m: string[] = [];
		for (let i = 7; i >= 0; i--) {
			const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
			m.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`);
		}
		return m;
	});
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') closeModal(); }} />

<h1 class="page-title">Transaktionen</h1>

<div class="row" style="margin-bottom:1rem;flex-wrap:wrap;align-items:center">
	<select bind:value={selectedAccount} style="width:auto;min-width:160px">
		{#each accList as a}<option value={a.id}>{a.name}</option>{/each}
	</select>
	<select bind:value={statusFilter} style="width:auto">
		<option value="">Alle</option>
		<option value="pending">Offen ({pendingCount})</option>
		<option value="posted">Bestätigt</option>
	</select>
	<select bind:value={monthFilter} style="width:auto">
		<option value="">Alle Monate</option>
		{#each months as m}<option value={m}>{m}</option>{/each}
	</select>
	<input type="search" bind:value={searchFilter} placeholder="Suchen…" style="width:140px;font-size:.82rem" />
	<button class="primary" onclick={() => showModal = true}><Icon name="plus" /> Neu</button>
	<button onclick={() => (document.querySelector('input[type=file]') as HTMLInputElement)?.click()}><Icon name="upload" /> CSV</button>
	<input type="file" accept=".csv" onchange={handleCsv} hidden />
</div>

{#if selected.size > 0}
	<div class="card" style="margin-bottom:1rem;display:flex;align-items:center;gap:.75rem;flex-wrap:wrap">
		<span style="color:var(--muted);font-size:.85rem">{selected.size} ausgewählt</span>
		<select bind:value={bulkCat} style="width:auto;min-width:140px">
			<option value="">Kategorie wählen…</option>
			{#each catList as c}<option value={c.id}>{c.name}</option>{/each}
		</select>
		<button class="primary small" onclick={bulkPost} disabled={selected.size === 0}>✓ Bestätigen</button>
		<button class="ghost small" onclick={() => selected = new Set()}>Abbrechen</button>
	</div>
{/if}

<div class="card" style="overflow-x:auto">
	<table style="min-width:700px">
		<thead><tr>
			<th style="width:30px"><input type="checkbox" onchange={toggleAll} checked={(txList||[]).filter(t => t.status === 'pending').length > 0 && selected.size === (txList||[]).filter(t => t.status === 'pending').length} /></th>
			<th>Datum</th><th>Beschreibung</th><th>Kategorie</th><th>Status</th><th>Betrag</th><th>⟳</th><th></th>
		</tr></thead>
		<tbody>
			{#each (txList || []) as tx}
				{@const cat = catById(tx.category_id)}
				<tr class:pending={tx.status === 'pending'}>
					<td><input type="checkbox" checked={selected.has(tx.id)} onchange={() => toggleSelect(tx.id)} /></td>
					<td style="color:var(--muted);white-space:nowrap">{tx.date}</td>
					<td>{tx.description || '—'}</td>
					<td>
						{#if tx.status === 'pending' && editId === tx.id}
							<select bind:value={quickCat} style="width:auto;font-size:.8rem;padding:.2rem .4rem">
								<option value="">—</option>
								{#each catList as c}<option value={c.id}>{c.name}</option>{/each}
							</select>
							<button class="primary small" style="margin-left:.3rem" onclick={() => quickPost(tx.id)}>✓</button>
						{:else if cat}
							<button class="ghost" style="padding:.1rem .4rem;font-size:.75rem;background:{cat.color}18;color:{cat.color};border:none;border-radius:4px" onclick={() => { if (tx.status === 'pending') { editId = tx.id; quickCat = tx.category_id; } }}>
								{cat.name} {tx.status === 'pending' ? '▾' : ''}
							</button>
						{:else}
							<button class="ghost" style="font-size:.75rem" onclick={() => { if (tx.status === 'pending') { editId = tx.id; quickCat = ''; } }}>
								zuordnen ▾
							</button>
						{/if}
					</td>
					<td>
						<span class="badge" style="background:{tx.status === 'pending' ? 'var(--gold)' : 'var(--green)'}18;color:{tx.status === 'pending' ? 'var(--gold)' : 'var(--green)'};font-size:.7rem">
							{tx.status === 'pending' ? 'offen' : '✓'}
							{#if tx.recurring_match_id} ⟳{/if}
						</span>
					</td>
					<td class="amount" class:income={tx.amount >= 0} class:expense={tx.amount < 0}>{fmtAmt(tx.amount)}</td>
					<td style="text-align:right;white-space:nowrap">
						<button class="ghost small" onclick={() => openEditTx(tx)} title="Bearbeiten">✎</button>
						<input type="checkbox" checked={!!tx.recurring_match_id} onchange={() => toggleExclude(tx.id)} title="Von Ausgaben ausschließen" style="margin-right:.4rem" />
						<button class="ghost small" onclick={() => delTx(tx.id)}><Icon name="trash" /></button>
					</td>
				</tr>
			{/each}
			{#if (txList||[]).length === 0}
				<tr><td colspan="8" style="color:var(--muted);text-align:center;padding:2.5rem">Keine Transaktionen</td></tr>
			{/if}
		</tbody>
	</table>
</div>

{#if showModal}
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={closeModal} role="presentation">
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>Neue Transaktion</h2>
		<div class="row" style="margin-bottom:.85rem">
			<button class:primary={isExpense} onclick={() => isExpense = !isExpense} style="font-size:.82rem">
				{isExpense ? 'Ausgabe' : 'Einnahme'}
			</button>
		</div>
		<div class="form-group">
			<label for="tx-cat">Kategorie</label>
			<select id="tx-cat" bind:value={selectedCategory}>
				<option value="">—</option>
				{#each catList.filter(c => isExpense ? !c.is_income : c.is_income) as c}
					<option value={c.id}>{c.name}</option>
				{/each}
			</select>
		</div>
		<div class="form-group">
			<label for="tx-amt">Betrag (€)</label>
			<input id="tx-amt" type="number" step="0.01" bind:value={amount} placeholder="42,50" />
		</div>
		<div class="form-group">
			<label for="tx-desc">Beschreibung</label>
			<input id="tx-desc" bind:value={description} placeholder="Einkauf bei Rewe" />
		</div>
		<div class="form-group">
			<label for="tx-date">Datum</label>
			<input id="tx-date" type="date" bind:value={date} />
		</div>
		<div class="modal-actions">
			<button onclick={closeModal}>Abbrechen</button>
			<button class="primary" onclick={createTx}>Hinzufügen</button>
		</div>
	</div>
</div>
{/if}

{#if showRecurringModal && recTx}
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={() => showRecurringModal = false} role="presentation">
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>Als Dauerauftrag merken</h2>
		<div class="form-group">
			<label>Beschreibung</label>
			<input value={recTx.description} disabled style="opacity:.7" />
		</div>
		<div class="form-group">
			<label>Betrag (€)</label>
			<input type="number" value={recTx.amount} disabled style="opacity:.7" />
		</div>
		<div class="row">
			<div class="form-group" style="flex:2">
				<label for="ri-kind">Intervall</label>
				<select id="ri-kind" bind:value={recInterval}>
					<option value="monthly">Monatlich</option>
					<option value="quarterly">Quartalsweise</option>
					<option value="yearly">Jährlich</option>
				</select>
			</div>
			<div class="form-group" style="flex:1">
				<label for="ri-n">Alle N</label>
				<input id="ri-n" type="number" min="1" bind:value={recIntervalN} />
			</div>
		</div>
		<div class="modal-actions">
			<button onclick={() => showRecurringModal = false}>Abbrechen</button>
			<button class="primary" onclick={saveRecurring}>Erstellen</button>
		</div>
	</div>
</div>
{/if}

{#if showEditTx}
<div class="modal-overlay" onclick={() => showEditTx = false} role="presentation">
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>Transaktion bearbeiten</h2>
		<div class="form-group"><label>Beschreibung</label><input bind:value={editTxDesc} /></div>
		<div class="form-group"><label>Betrag (€)</label><input type="number" step="0.01" bind:value={editTxAmount} /></div>
		<div class="form-group"><label>Datum</label><input type="date" bind:value={editTxDate} /></div>
		<div class="form-group">
			<label>Kategorie</label>
			<select bind:value={editTxCat}>
				{#each catList as c}<option value={c.id}>{c.name}</option>{/each}
			</select>
		</div>
		<div class="modal-actions">
			<button onclick={() => showEditTx = false}>Abbrechen</button>
			<button class="primary" onclick={saveEditTx}>Speichern</button>
		</div>
	</div>
</div>
{/if}
