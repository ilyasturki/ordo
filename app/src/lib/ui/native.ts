import { App } from '@capacitor/app';
import { goto } from '$app/navigation';
import { page } from '$app/state';
import { closeTopOverlay } from '$lib/ui/overlays';

const TEXT = 'input:not([type="checkbox"]):not([type="radio"]), textarea, [contenteditable]';

function refuse(e: Event) {
	if (!(e.target instanceof Element && e.target.closest(TEXT))) {
		e.preventDefault();
	}
}

/** What an Android app does and a browser tab doesn't: no long-press menus, and back closes what's on top. */
export function wireNative(): () => void {
	if (!import.meta.env.APP_BUILD) {
		return () => undefined;
	}
	document.documentElement.classList.add('native');
	const listeners = new AbortController();
	document.addEventListener('contextmenu', refuse, { signal: listeners.signal });
	document.addEventListener('dragstart', refuse, { signal: listeners.signal });
	const back = App.addListener('backButton', () => {
		if (closeTopOverlay()) {
			return;
		}
		if (page.url.pathname !== '/') {
			if (history.length > 1) {
				history.back();
			} else {
				void goto('/', { replaceState: true });
			}
			return;
		}
		void App.minimizeApp();
	});
	return () => {
		listeners.abort();
		void back.then((h) => h.remove());
	};
}
