import type { Reroute } from '@sveltejs/kit';
import { route } from '$lib/paths';

export const reroute: Reroute = ({ url }) => route(url.pathname);
