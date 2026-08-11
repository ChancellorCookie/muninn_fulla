<script lang="ts">
	let { segments, size = 180 }: {
		segments: { color: string; value: number; label: string }[];
		size?: number;
	} = $props();

	const total = $derived(segments.reduce((s, seg) => s + Math.abs(seg.value), 0) || 1);
	const cx = size / 2, cy = size / 2, r = size * 0.38, ring = size * 0.15;

	function path(seg: typeof segments[0], i: number) {
		const startAngle = segments.slice(0, i).reduce((s, seg) => s + (Math.abs(seg.value) / total) * 2 * Math.PI, -Math.PI / 2);
		const sliceAngle = (Math.abs(seg.value) / total) * 2 * Math.PI;
		const endAngle = startAngle + sliceAngle;
		const x1 = cx + r * Math.cos(startAngle), y1 = cy + r * Math.sin(startAngle);
		const x2 = cx + r * Math.cos(endAngle), y2 = cy + r * Math.sin(endAngle);
		const largeArc = sliceAngle > Math.PI ? 1 : 0;
		return `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`;
	}
</script>

<svg width={size} height={size} viewBox="0 0 {size} {size}" style="display:block">
	{#each segments as seg, i}
		<path d={path(seg, i)} fill={seg.color} stroke="var(--bg)" stroke-width="1.5">
			<title>{seg.label}: {Math.abs(seg.value).toFixed(2)} €</title>
		</path>
	{/each}
	<circle cx={cx} cy={cy} r={r - ring} fill="var(--bg)" />
	<text x={cx} y={cy - 6} text-anchor="middle" fill="var(--text)" font-size="12" font-weight="700">
		{total.toFixed(0)} €
	</text>
	<text x={cx} y={cy + 10} text-anchor="middle" fill="var(--muted)" font-size="10">
		erwartet
	</text>
</svg>
