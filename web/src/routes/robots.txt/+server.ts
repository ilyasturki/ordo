import { origin } from '$lib/landing/origin';

export const prerender = true;

export function GET() {
	return new Response(`User-agent: *\nDisallow: /app\nSitemap: ${origin}sitemap.xml\n`, {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' }
	});
}
