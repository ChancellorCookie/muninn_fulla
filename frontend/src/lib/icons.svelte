<script lang="ts">
	let { name, color = 'currentColor' }: { name: string; color?: string } = $props();
</script>

{#if name === 'chart'}
	<svg class="icon" viewBox="0 0 24 24"><path d="M18 20V10M12 20V4M6 20v-6" stroke={color}/></svg>
{:else if name === 'list'}
	<svg class="icon" viewBox="0 0 24 24">
		<line x1="8" y1="6" x2="21" y2="6" stroke={color}/><line x1="8" y1="12" x2="21" y2="12" stroke={color}/>
		<line x1="8" y1="18" x2="21" y2="18" stroke={color}/><line x1="3" y1="6" x2="3.01" y2="6" stroke={color}/>
		<line x1="3" y1="12" x2="3.01" y2="12" stroke={color}/><line x1="3" y1="18" x2="3.01" y2="18" stroke={color}/>
	</svg>
{:else if name === 'tag'}
	<svg class="icon" viewBox="0 0 24 24"><path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82zM7 7h.01" stroke={color}/></svg>
{:else if name === 'repeat'}
	<svg class="icon" viewBox="0 0 24 24">
		<polyline points="17 1 21 5 17 9" stroke={color}/>
		<path d="M3 11V9a4 4 0 014-4h14" stroke={color}/>
		<polyline points="7 23 3 19 7 15" stroke={color}/>
		<path d="M21 13v2a4 4 0 01-4 4H3" stroke={color}/>
	</svg>
{:else if name === 'plus'}
	<svg class="icon" viewBox="0 0 24 24"><line x1="12" y1="5" x2="12" y2="19" stroke={color}/><line x1="5" y1="12" x2="19" y2="12" stroke={color}/></svg>
{:else if name === 'trash'}
	<svg class="icon" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6" stroke={color}/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" stroke={color}/></svg>
{:else if name === 'upload'}
	<svg class="icon" viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" stroke={color}/><polyline points="17 8 12 3 7 8" stroke={color}/><line x1="12" y1="3" x2="12" y2="15" stroke={color}/></svg>
{:else if name === 'settings'}
	<svg class="icon" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3" stroke={color}/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" stroke={color}/></svg>
{/if}
