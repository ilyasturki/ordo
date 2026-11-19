import type { HandleClientError } from '@sveltejs/kit/hooks';

// Kit 3 reports a hovered link's failed preload here: offline, that is any route module not fetched yet.
export const handleError: HandleClientError = ({ kind, error }) => {
	if (kind === 'unknown' && !(error instanceof TypeError && !navigator.onLine)) console.error(error);
};
