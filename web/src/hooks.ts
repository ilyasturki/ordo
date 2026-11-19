import type { Reroute } from '@sveltejs/kit/hooks';
import { route } from '#lib/paths.ts';

export const reroute: Reroute = ({ url }) => route(url.pathname);
