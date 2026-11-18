<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Local } from '$lib/local.svelte';
	import { app } from '$lib/paths';
	import { exportProject, importProject } from './actions';
	import IconButton from './IconButton.svelte';
	import Menu from './Menu.svelte';
	import MenuItem from './MenuItem.svelte';
	import { ui } from './ui.svelte';

	let { local, pid }: { local: Local; pid: string } = $props();

	let open = $state(false);
	let anchor: HTMLButtonElement | undefined = $state();
</script>

<IconButton bind:el={anchor} icon="more" label="More options" onclick={() => (open = !open)} aria-haspopup="menu" aria-expanded={open} />
<Menu bind:open {anchor} align="end" label="Project">
	<MenuItem icon="edit" label="Rename project" onclick={() => (ui.sheet = { kind: 'project', pid })} />
	<MenuItem icon="group" label="Group" onclick={() => (ui.sheet = { kind: 'group', pid })} />
	<MenuItem icon="newFolder" label="New category" keys="A" onclick={() => (ui.sheet = { kind: 'newCategory', pid, parent: '' })} />
	<MenuItem icon="categories" label="Categories" onclick={() => void goto(app(`/p/${pid}/categories`))} />
	<MenuItem icon="download" label="Export project" onclick={() => exportProject(local, pid)} />
	<MenuItem icon="upload" label="Import a project" onclick={() => importProject(local)} />
	<MenuItem icon="delete" label="Delete project" danger onclick={() => (ui.sheet = { kind: 'deleteProject', pid })} />
	<hr class="my-2 border-outline-variant" />
	<MenuItem icon="keyboard" label="Keyboard shortcuts" keys="?" onclick={() => (ui.sheet = { kind: 'help' })} />
</Menu>
