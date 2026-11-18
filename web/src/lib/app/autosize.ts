import type { Action } from 'svelte/action';

/** Grows a textarea with its text: Firefox has no field-sizing. */
export const autosize: Action<HTMLTextAreaElement, string> = (el) => {
	const fit = () => {
		el.style.height = 'auto';
		el.style.height = `${el.scrollHeight}px`;
	};
	fit();
	el.addEventListener('input', fit);
	const observer = new ResizeObserver(() => requestAnimationFrame(fit));
	observer.observe(el.parentElement ?? el);
	return {
		update: () => requestAnimationFrame(fit),
		destroy: () => {
			el.removeEventListener('input', fit);
			observer.disconnect();
		}
	};
};
