<script lang="ts">
	interface Bar {
		label: string;
		color: string;
		recurring: number;
		extra: number;
	}

	let { data }: { data: Bar[] } = $props();

	const colW = 28;
	const colGap = 14;
	const segGap = 2;
	const chartH = 180;
	const bottomPad = 50;
	const topPad = 10;
	const leftPad = 45;
	const totalW = $derived(data.length * (colW + colGap) + leftPad + 20);
	const totalH = $derived(chartH + bottomPad + topPad);

	const maxVal = $derived(Math.max(1, ...data.map(d => Math.abs(d.recurring) + Math.abs(d.extra))));
</script>

<div style="font-size:.7rem;color:var(--muted);display:flex;gap:1rem;justify-content:flex-end;margin-bottom:.25rem">
	<span><span style="display:inline-block;width:10px;height:10px;background:#888;border-radius:2px;opacity:0.85;margin-right:4px;vertical-align:middle"></span> laufend</span>
	<span><span style="display:inline-block;width:10px;height:10px;background:#888;border-radius:2px;opacity:0.3;margin-right:4px;vertical-align:middle"></span> zusätzlich</span>
</div>

<svg width="100%" viewBox="0 0 {totalW} {totalH}" style="max-width:{totalW}px">
	<!-- Y-axis grid lines -->
	{#each [0, 0.25, 0.5, 0.75, 1] as tick}
		{@const y = chartH - tick * chartH + topPad}
		<line x1={leftPad} y1={y} x2={totalW - 10} y2={y} stroke="var(--border,#333)" stroke-width="0.5" />
		<text x={leftPad - 4} y={y + 3} text-anchor="end" fill="var(--muted,#888)" font-size="9">
			{Math.round(maxVal * tick)}
		</text>
	{/each}

	{#each data as bar, i}
		{@const x = leftPad + i * (colW + colGap)}
		{@const recH = Math.abs(bar.recurring) / maxVal * chartH}
		{@const extH = Math.abs(bar.extra) / maxVal * chartH}
		{@const totalH_ = recH + extH}

		<!-- Extra (top segment, lighter) -->
		{#if extH > 0.5}
			<rect x={x} y={chartH + topPad - recH - extH} width={colW} height={extH} fill={bar.color} opacity="0.3" rx="2" />
		{/if}

		<!-- Recurring (bottom segment, solid) -->
		<rect x={x} y={chartH + topPad - recH} width={colW} height={recH} fill={bar.color} opacity="0.85" rx="2" />

		<!-- Category label (angled) -->
		<g transform="translate({x + colW/2}, {chartH + topPad + 8})">
			<text transform="rotate(-35)" text-anchor="end" fill="var(--muted,#aaa)" font-size="9">
				{bar.label}
			</text>
		</g>
	{/each}
</svg>
