import type { HandleClientError } from '@sveltejs/kit/hooks';

// Chromium, Firefox and WebKit's words for a route module that didn't arrive.
const missedModule = /dynamically imported module|Importing a module script failed/;

// Kit 3 hands a hovered link's failed preload here too: offline, or when the page unloads mid-import.
export const handleError: HandleClientError = ({ kind, error }) => {
	if (kind === 'unknown' && !(error instanceof TypeError && missedModule.test(error.message))) console.error(error);
};
