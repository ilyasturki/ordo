import { origin } from '$lib/landing/origin';

export const prerender = true;

export function GET() {
	return new Response(
		`<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>${origin}</loc></url></urlset>\n`,
		{ headers: { 'Content-Type': 'application/xml; charset=utf-8' } }
	);
}
