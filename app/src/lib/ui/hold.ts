import type { Action } from 'svelte/action';

const HOLD_MS = 450;
const SLOP_PX = 10;

/** Long-press: calls onhold once the finger rests, and swallows the click its release would make. */
export const hold: Action<HTMLElement, (e: PointerEvent) => void> = (node, onhold) => {
	let timer: ReturnType<typeof setTimeout> | undefined;
	let start: { x: number; y: number } | null = null;
	let held = false;
	let handler = onhold;

	function cancel() {
		clearTimeout(timer);
		start = null;
	}

	function down(e: PointerEvent) {
		if (e.button !== 0) {
			return;
		}
		held = false;
		start = { x: e.clientX, y: e.clientY };
		timer = setTimeout(() => {
			held = true;
			start = null;
			navigator.vibrate?.(10);
			handler(e);
		}, HOLD_MS);
	}

	function move(e: PointerEvent) {
		if (start !== null && Math.hypot(e.clientX - start.x, e.clientY - start.y) > SLOP_PX) {
			cancel();
		}
	}

	function click(e: MouseEvent) {
		if (held) {
			held = false;
			e.preventDefault();
			e.stopPropagation();
		}
	}

	node.addEventListener('pointerdown', down);
	node.addEventListener('pointermove', move);
	node.addEventListener('pointerup', cancel);
	node.addEventListener('pointercancel', cancel);
	node.addEventListener('click', click, true);
	return {
		update(next) {
			handler = next;
		},
		destroy() {
			cancel();
			node.removeEventListener('pointerdown', down);
			node.removeEventListener('pointermove', move);
			node.removeEventListener('pointerup', cancel);
			node.removeEventListener('pointercancel', cancel);
			node.removeEventListener('click', click, true);
		}
	};
};
