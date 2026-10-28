import type { Action } from 'svelte/action';

/** A Material ripple from where the finger landed, on an element styled `state-layer`. */
export const ripple: Action<HTMLElement> = (node) => {
	function down(e: PointerEvent) {
		if (e.button !== 0) {
			return;
		}
		const box = node.getBoundingClientRect();
		const x = e.clientX - box.left;
		const y = e.clientY - box.top;
		const r = Math.hypot(Math.max(x, box.width - x), Math.max(y, box.height - y));
		const drop = document.createElement('span');
		drop.className = 'ripple-drop';
		drop.ariaHidden = 'true';
		drop.style.left = `${x - r}px`;
		drop.style.top = `${y - r}px`;
		drop.style.width = drop.style.height = `${r * 2}px`;
		node.append(drop);
		const release = () => {
			drop.classList.add('fading');
			setTimeout(() => drop.remove(), 300);
			window.removeEventListener('pointerup', release);
			window.removeEventListener('pointercancel', release);
		};
		window.addEventListener('pointerup', release);
		window.addEventListener('pointercancel', release);
	}
	node.addEventListener('pointerdown', down);
	return {
		destroy() {
			node.removeEventListener('pointerdown', down);
		}
	};
};
