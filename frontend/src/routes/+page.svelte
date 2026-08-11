<script lang="ts">
	import { onMount } from 'svelte';
	import Donut from '$lib/Donut.svelte';
	import Bars from '$lib/Bars.svelte';
	import { accounts as accApi, recurring, categories } from '$lib/api';
	import type { Account, ForecastResponse, MonthSummary, Category } from '$lib/types';

	let accountList = $state<Account[]>([]);
	let forecast = $state<ForecastResponse | null>(null);
	let summary = $state<MonthSummary | null>(null);
	let txByCat = $state<Map<string,{category_id:string;items:{id:string;name:string;amount:number}[];total:number}>>(new Map());
	let catList = $state<Category[]>([]);
	let month = $state(new Date().toISOString().slice(0, 7));
	let year = $state(new Date().getFullYear());
	let annual = $state<any>(null);
	let loading = $state(true);

	// Edit modal for recurring
	let showEdit = $state(false);
	let editRecurring: any = $state(null);
	let editAmount = $state('');
	let editDesc = $state('');
	let editCatId = $state('');
	let editInterval = $state('monthly');
	let editIntervalN = $state(1);

	const months = $derived.by(() => {
		const now = new Date(); const m: string[] = [];
		for (let i = 11; i >= 0; i--) {
			const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
			m.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`);
		}
		return m;
	});
	const totalBalance = $derived(accountList.reduce((s, a) => s + a.balance, 0));
	const dashboardSaldo = $derived((forecast?.income||0) + (summary?.income||0) + (forecast?.expenses||0) + (summary?.expenses||0));

	onMount(async () => {
		accountList = await accApi.list();
		catList = await categories.list();
		await loadData();
		loading = false;
	});

	async function loadData() {
		try {
			const [f, s, tx] = await Promise.all([
				fetch(`/api/forecast?month=${month}`).then(r => r.json()),
				fetch(`/api/summary?month=${month}`).then(r => r.json()),
				fetch(`/api/transactions?month=${month}&status=posted`).then(r => r.json()),
			]);
			forecast = f; summary = s;
			// Group actual transactions by category
			txByCat = new Map();
			for (const t of tx || []) {
				if (t.recurring_match_id) continue; // skip excluded
				const cid = t.category_id;
				if (!txByCat.has(cid)) txByCat.set(cid, { category_id: cid, items: [], total: 0 });
				const g = txByCat.get(cid)!;
				g.items.push({ id: t.id, name: t.description, amount: t.amount });
				g.total += t.amount;
			}
		} catch(e) { console.error(e); }
	}
	$effect(() => { month; loadData(); });
	$effect(() => { year; loadAnnual(); });

	async function loadAnnual() {
		try {
			const r = await fetch(`/api/annual?year=${year}`).then(r => r.json());
			annual = r;
		} catch(e) { console.error(e); }
	}

	const chartSegments = $derived((forecast?.by_cat || [])
		.filter(c => c.total !== 0)
		.map(c => ({ color: c.color, value: c.total, label: c.category_name }))
	);

	const barData = $derived.by(() => {
		const cats = new Map<string, { label: string; color: string; recurring: number; extra: number }>();
		// Recurring
		for (const col of forecast?.by_cat || []) {
			if (col.total >= 0) continue; // skip income
			const c = cats.get(col.category_name) || { label: col.category_name, color: col.color, recurring: 0, extra: 0 };
			c.recurring += col.total;
			cats.set(col.category_name, c);
		}
		// One-time
		for (const [cid, group] of txByCat) {
			const cat = catList.find(c => c.id === cid);
			const name = cat?.name || cid;
			const c = cats.get(name) || { label: name, color: cat?.color || '#888', recurring: 0, extra: 0 };
			c.extra += group.total;
			cats.set(name, c);
		}
		return [...cats.values()].filter(d => d.recurring + d.extra < 0).sort((a, b) => (a.recurring + a.extra) - (b.recurring + b.extra));
	});

	function openEdit(item: any) {
		editRecurring = item;
		editAmount = String(Math.abs(item.amount));
		editDesc = item.name;
		editCatId = forecast?.by_cat.find(c => c.items.some(i => i.id === item.id))?.category_id || '';
		editInterval = 'monthly'; editIntervalN = 1;
		// Find existing recurring to pre-fill
		fetch(`/api/recurring/${item.id}`).then(r => r.json()).then(rt => {
			if (rt && rt.interval_kind) { editInterval = rt.interval_kind; editIntervalN = rt.interval_n || 1; }
		}).catch(() => {});
		showEdit = true;
	}

	async function saveEdit() {
		if (!editRecurring) return;
		const amt = parseFloat(editAmount);
		const catForecast = forecast?.by_cat.find(c => c.items.some(i => i.id === editRecurring.id));
		const isIncome = catForecast && catForecast.total > 0;
		await recurring.update(editRecurring.id, {
			account_id: accountList[0]?.id || '',
			category_id: editCatId,
			amount: isIncome ? Math.abs(amt) : -Math.abs(amt),
			description: editDesc,
			interval_kind: editInterval,
			interval_n: editIntervalN,
			next_due: '',
		});
		showEdit = false;
		await loadData();
	}

	async function delEdit() {
		if (!editRecurring || !confirm('Dauerauftrag löschen?')) return;
		await recurring.del(editRecurring.id);
		showEdit = false;
		await loadData();
	}
</script>

{#if loading}<p style="color:var(--muted);padding:2rem">Lade…</p>
{:else}
	<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1.5rem">
		<h1 style="margin:0;font-size:1.3rem">Monatliche Ausgaben</h1>
		<select bind:value={month} style="width:auto;font-size:.85rem">
			{#each months as m}<option value={m}>{m}</option>{/each}
		</select>
	</div>

	{@const recSaldo = (forecast?.income||0) + (forecast?.expenses||0)}
	{@const gesSaldo = (forecast?.income||0) + (summary?.income||0) + (forecast?.expenses||0) + (summary?.expenses||0)}

	<!-- Row 1: Laufende Kosten + Recurring-Saldo -->
	<div style="display:grid;grid-template-columns:3fr 1fr;gap:1rem;margin-bottom:1rem;align-items:stretch">
		<div class="card" style="overflow-x:auto">
			<h3 style="font-size:.82rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em;margin-bottom:.75rem">Laufende Kosten</h3>
			{#if forecast && forecast.by_cat.length > 0}
				<div style="display:flex;gap:1.5rem;min-width:fit-content">
					{#each forecast.by_cat.filter(c => c.total !== 0) as col}
						<div style="min-width:130px;max-width:180px">
							<div style="display:flex;align-items:center;gap:.4rem;margin-bottom:.5rem;padding-bottom:.35rem;border-bottom:2px solid {col.color}">
								<span style="font-weight:700;font-size:.78rem">{col.category_name}</span>
							</div>
							{#each col.items as item}
								<div style="padding:.15rem 0;display:flex;justify-content:space-between;gap:.5rem;font-size:.78rem">
									<span style="white-space:nowrap;cursor:pointer" onclick={() => openEdit(item)} title="Klicken zum Bearbeiten">{item.name}</span>
									<span style="color:var(--muted);white-space:nowrap">{item.amount >= 0 ? '+' : ''}{item.amount.toFixed(0)}€</span>
								</div>
							{/each}
							<div style="margin-top:.35rem;padding-top:.35rem;border-top:1px solid var(--border);font-weight:700;font-size:.78rem;color:{col.total >= 0 ? '#4a7c59' : '#c45c5c'}">
								{col.total >= 0 ? '+' : ''}{col.total.toFixed(0)} €
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
		<div class="card" style="padding:1.25rem;text-align:center;display:flex;flex-direction:column;justify-content:center;gap:.6rem">
			<div style="font-size:.75rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em">Saldo laufend</div>
			<div style="font-size:.78rem"><span style="color:var(--muted)">Einkommen</span> <span class="positive" style="font-weight:700">{forecast?.income?.toFixed(0)||'0'} €</span></div>
			<div style="font-size:.78rem"><span style="color:var(--muted)">Kosten</span> <span class="negative" style="font-weight:700">{Math.abs(forecast?.expenses||0).toFixed(0)} €</span></div>
			<div style="border-top:1px solid var(--border);padding-top:.5rem;font-size:1.1rem;font-weight:700" class:positive={recSaldo>=0} class:negative={recSaldo<0}>
				{recSaldo>=0?'+':''}{recSaldo.toFixed(0)} €
			</div>
		</div>
	</div>

	<!-- Row 2: Ausgaben + Gesamt-Saldo -->
	<div style="display:grid;grid-template-columns:3fr 1fr;gap:1rem;margin-bottom:1.5rem;align-items:stretch">
		<div class="card" style="overflow-x:auto">
			<h3 style="font-size:.82rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em;margin-bottom:.75rem">Zusätzliche Ausgaben / Einnahmen</h3>
			{#if txByCat.size > 0}
				<div style="display:flex;gap:1.5rem;min-width:fit-content">
					{#each [...txByCat.values()].filter(g => g.total !== 0) as col}
						{@const cat = catList.find(c => c.id === col.category_id)}
						<div style="min-width:130px;max-width:180px">
							<div style="display:flex;align-items:center;gap:.4rem;margin-bottom:.5rem;padding-bottom:.35rem;border-bottom:2px solid {cat?.color || '#888'}">
								<span style="font-weight:700;font-size:.78rem">{cat?.name || '—'}</span>
							</div>
							{#each col.items as item}
								<div style="padding:.15rem 0;display:flex;justify-content:space-between;gap:.5rem;font-size:.78rem">
									<span style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:110px" title={item.name}>{item.name}</span>
									<span style="color:var(--muted);white-space:nowrap">{item.amount >= 0 ? '+' : ''}{item.amount.toFixed(0)}€</span>
								</div>
							{/each}
							<div style="margin-top:.35rem;padding-top:.35rem;border-top:1px solid var(--border);font-weight:700;font-size:.78rem;color:{col.total >= 0 ? '#4a7c59' : '#c45c5c'}">
								{col.total >= 0 ? '+' : ''}{col.total.toFixed(0)} €
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<p style="color:var(--muted);font-size:.84rem;padding:.5rem 0">Keine Buchungen in diesem Monat.</p>
			{/if}
		</div>
		<div class="card" style="padding:1.25rem;text-align:center;display:flex;flex-direction:column;justify-content:center;gap:.5rem">
			<div style="font-size:.75rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em">Gesamtsaldo</div>
			<div style="font-size:.75rem"><span style="color:var(--muted)">Einkommen</span> <span style="font-weight:700;color:#4a7c59">{forecast?.income?.toFixed(0)||'0'} €</span></div>
			<div style="font-size:.75rem"><span style="color:var(--muted)">Einnahmen</span> <span style="font-weight:700;color:#4a7c59">{summary?.income?.toFixed(0)||'0'} €</span></div>
			<div style="font-size:.75rem"><span style="color:var(--muted)">lauf. Kosten</span> <span style="font-weight:700;color:#c45c5c">{Math.abs(forecast?.expenses||0).toFixed(0)} €</span></div>
			<div style="font-size:.75rem"><span style="color:var(--muted)">Ausgaben</span> <span style="font-weight:700;color:#c45c5c">{Math.abs(summary?.expenses||0).toFixed(0)} €</span></div>
			<div style="border-top:1px solid var(--border);margin-top:.25rem;padding-top:.5rem;font-size:1.15rem;font-weight:700" class:positive={gesSaldo>=0} class:negative={gesSaldo<0}>
				{gesSaldo>=0?'+':''}{gesSaldo.toFixed(0)} €
			</div>
		</div>
	</div>

	<!-- Row 3: Bar chart -->
	{#if barData.length > 0}
		<div class="card" style="padding:1rem">
			<h3 style="font-size:.82rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em;margin-bottom:.75rem">Monatliche Ausgaben</h3>
			<Bars data={barData} />
		</div>
	{/if}

	<!-- Row 4: Annual overview -->
	<div class="card" style="padding:1rem;margin-top:1.5rem">
		<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:.75rem">
			<h3 style="font-size:.82rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em;margin:0">Jahresübersicht</h3>
			<select bind:value={year} style="width:auto;font-size:.82rem">
				<option value={2026}>2026</option>
				<option value={2025}>2025</option>
			</select>
		</div>
		{#if annual?.months?.length > 0}
			<div style="overflow-x:auto">
				<table style="min-width:600px;font-size:.8rem">
					<thead><tr>
						<th>Monat</th><th style="color:#4a7c59">Einkommen</th><th style="color:#4a7c59">+ Einnahmen</th>
						<th style="color:#c45c5c">Kosten</th><th style="color:#c45c5c">+ Ausgaben</th><th>Saldo</th>
					</tr></thead>
					<tbody>
						{#each annual.months as m}
							{@const saldo = m.income + m.expenses + m.extra_income + m.extra_expenses}
							<tr>
								<td style="font-weight:600">{m.month}</td>
								<td style="color:var(--muted)">{m.income.toFixed(0)} €</td>
								<td style="color:var(--muted)">{m.extra_income.toFixed(0)} €</td>
								<td style="color:var(--muted)">{Math.abs(m.expenses).toFixed(0)} €</td>
								<td style="color:var(--muted)">{Math.abs(m.extra_expenses).toFixed(0)} €</td>
								<td style="font-weight:700" class:positive={saldo>=0} class:negative={saldo<0}>{saldo>=0?'+':''}{saldo.toFixed(0)} €</td>
							</tr>
						{/each}
					</tbody>
					<tfoot>
						<tr style="border-top:2px solid var(--border);font-weight:700">
							<td>Gesamt</td>
							<td>{annual.totals.income.toFixed(0)} €</td>
							<td>{annual.totals.extra_income.toFixed(0)} €</td>
							<td>{Math.abs(annual.totals.expenses).toFixed(0)} €</td>
							<td>{Math.abs(annual.totals.extra_expenses).toFixed(0)} €</td>
							<td class:positive={annual.totals.balance>=0} class:negative={annual.totals.balance<0}>
								{annual.totals.balance>=0?'+':''}{annual.totals.balance.toFixed(0)} €
							</td>
						</tr>
					</tfoot>
				</table>
			</div>
		{:else}
			<p style="color:var(--muted);font-size:.84rem">Keine Daten für {year}.</p>
		{/if}
	</div>
{/if}
<!-- Edit Modal -->
{#if showEdit && editRecurring}
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={() => showEdit = false} role="presentation">
	<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
		<h2>Dauerauftrag bearbeiten</h2>
		<div class="form-group"><label>Name</label><input bind:value={editDesc} /></div>
		<div class="form-group">
			<label>Betrag (€)</label>
			<input type="number" step="0.01" bind:value={editAmount} />
		</div>
		<div class="form-group">
			<label>Kategorie</label>
			<select bind:value={editCatId}>
				{#each catList as c}<option value={c.id}>{c.name}</option>{/each}
			</select>
		</div>
		<div class="row">
			<div class="form-group" style="flex:2">
				<label>Intervall</label>
				<select bind:value={editInterval}>
					<option value="monthly">Monatlich</option>
					<option value="quarterly">Quartalsweise</option>
					<option value="yearly">Jährlich</option>
				</select>
			</div>
			<div class="form-group" style="flex:1">
				<label>Alle N</label>
				<input type="number" min="1" bind:value={editIntervalN} />
			</div>
		</div>
		<div class="modal-actions">
			<button onclick={() => showEdit = false}>Abbrechen</button>
			<button class="danger" onclick={delEdit} style="margin-right:auto">Löschen</button>
			<button class="primary" onclick={saveEdit}>Speichern</button>
		</div>
	</div>
</div>
{/if}
