<script lang="ts" module>
	const paths = {
		search:
			'M15.5 14h-.79l-.28-.27A6.47 6.47 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z',
		filter: 'M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z',
		more: 'M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z',
		add: 'M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z',
		less: 'M12 8l-6 6 1.41 1.41L12 10.83l4.59 4.58L18 14z',
		drop: 'M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6z',
		up: 'M7.41 15.41 12 10.83l4.59 4.58L18 14l-6-6-6 6z',
		down: 'M7.41 8.59 12 13.17l4.59-4.58L18 10l-6 6-6-6z',
		upup: 'M7.41 18.41 12 13.83l4.59 4.58L18 17l-6-6-6 6zM7.41 11.41 12 6.83l4.59 4.58L18 10l-6-6-6 6z',
		subtask: 'M19 15l-6 6-1.42-1.42L15.17 16H4V4h2v10h9.17l-3.59-3.58L13 9l6 6z',
		photo:
			'M19 7v2.99s-1.99.01-2 0V7h-3s.01-1.99 0-2h3V2h2v3h3v2h-3zm-3 4V8h-3V5H5c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2v-8h-3zM5 19l3-4 2 3 3-4 4 5H5z',
		check: 'M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z',
		copy: 'M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z',
		doc: 'M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm4 18H6V4h7v5h5v11z',
		logo: ''
	};

	export type IconName = keyof typeof paths | 'todo' | 'prog' | 'done';
</script>

<script lang="ts">
	let { name, class: className = '' }: { name: IconName; class?: string } = $props();
</script>

{#if name === 'logo'}
	<svg class={className} viewBox="18 18 72 72" aria-hidden="true">
		<rect x="18" y="18" width="72" height="72" rx="18" fill="#3949AB" />
		<path
			fill="#FFFFFF"
			d="M32.5,63.5h19a5.5,5.5 0 0 1 0 11h-19a5.5,5.5 0 0 1 0-11zM44.5,48.5h19a5.5,5.5 0 0 1 0 11h-19a5.5,5.5 0 0 1 0-11zM56.5,33.5h19a5.5,5.5 0 0 1 0 11h-19a5.5,5.5 0 0 1 0-11z"
		/>
	</svg>
{:else}
	<svg class={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
		{#if name === 'todo' || name === 'prog'}
			<circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
			{#if name === 'prog'}<path d="M12 6a6 6 0 0 0 0 12z" />{/if}
		{:else if name === 'done'}
			<circle cx="12" cy="12" r="10" />
			<path d="M10.2 16.2 6.6 12.6l1.4-1.4 2.2 2.2 5.8-5.8 1.4 1.4z" fill="var(--color-surface)" />
		{:else}
			<path d={paths[name]} />
		{/if}
	</svg>
{/if}
