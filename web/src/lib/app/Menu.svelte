<script lang="ts">
	import type { Snippet } from 'svelte';

	// Anchored to what opened it, in the top layer, so it also opens from inside a modal sheet.
	let {
		open = $bindable(false),
		anchor,
		align = 'start',
		label,
		onclose,
		children
	}: { open: boolean; anchor: HTMLElement | undefined; align?: 'start' | 'end'; label: string; onclose?: () => void; children: Snippet } = $props();

	let el: HTMLElement;

	function place() {
		if (!anchor) return;
		const a = anchor.getBoundingClientRect();
		const m = el.getBoundingClientRect();
		let top = a.bottom + 4;
		if (top + m.height > innerHeight - 8) top = Math.max(8, a.top - m.height - 4);
		const left = align === 'end' ? a.right - m.width : a.left;
		el.style.top = `${top}px`;
		el.style.left = `${Math.min(Math.max(8, left), innerWidth - m.width - 8)}px`;
	}

	function close(refocus: boolean) {
		open = false;
		onclose?.();
		if (refocus) anchor?.focus();
	}

	$effect(() => {
		if (!open) {
			if (el.matches(':popover-open')) el.hidePopover();
			return;
		}
		el.showPopover();
		place();
		(el.querySelector<HTMLElement>('[aria-checked="true"]') ?? el.querySelector<HTMLElement>('[role^="menuitem"]'))?.focus();
		const away = (e: PointerEvent) => {
			const target = e.target as Node;
			if (!el.contains(target) && !anchor?.contains(target)) close(false);
		};
		const gone = () => close(false);
		addEventListener('pointerdown', away, true);
		addEventListener('resize', gone);
		return () => {
			removeEventListener('pointerdown', away, true);
			removeEventListener('resize', gone);
		};
	});

	function keydown(e: KeyboardEvent) {
		const items = [...el.querySelectorAll<HTMLElement>('[role^="menuitem"]')];
		const at = items.indexOf(document.activeElement as HTMLElement);
		const go = (i: number) => items[(i + items.length) % items.length]?.focus();
		e.stopPropagation();
		if (e.key === 'ArrowDown' || e.key === 'j') go(at + 1);
		else if (e.key === 'ArrowUp' || e.key === 'k') go(at - 1);
		else if (e.key === 'Home') go(0);
		else if (e.key === 'End') go(-1);
		else if (e.key === 'Escape') close(true);
		else if (e.key === 'Tab') close(false);
		else return;
		e.preventDefault();
	}

	function click(e: MouseEvent) {
		if ((e.target as HTMLElement).closest('[role^="menuitem"]')) close(false);
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div
	bind:this={el}
	popover="manual"
	role="menu"
	tabindex="-1"
	aria-label={label}
	onkeydown={keydown}
	onclick={click}
	class="m-0 max-h-[min(28rem,80dvh)] min-w-48 overflow-y-auto rounded-xs bg-surface-container py-2 text-on-surface shadow-lg"
>
	{@render children()}
</div>
