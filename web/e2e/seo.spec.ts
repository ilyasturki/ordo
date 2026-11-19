import { expect, test } from './fixtures';

test('the site build serves a robots.txt that keeps the app out and names the sitemap', async ({ request, build }) => {
	test.skip(build !== 'site', 'the private build has no crawlers to guide');
	const res = await request.get('/robots.txt');
	expect(res.status()).toBe(200);
	expect(res.headers()['content-type']).toMatch(/^text\/plain/);
	const robots = await res.text();
	expect(robots).toMatch(/^User-agent: \*$/m);
	expect(robots).toMatch(/^Disallow: \/app$/m);
	expect(robots).not.toMatch(/^Disallow:\s*\/\s*$/m);
	expect(robots).toMatch(/^Sitemap: https:\/\/ordo\.ilyasturki\.com\/sitemap\.xml$/m);
});

test('the site build serves a sitemap.xml listing the landing', async ({ request, build }) => {
	test.skip(build !== 'site', 'the private build has no crawlers to guide');
	const res = await request.get('/sitemap.xml');
	expect(res.status()).toBe(200);
	expect(res.headers()['content-type']).toMatch(/xml/);
	const xml = await res.text();
	expect(xml).toMatch(/<urlset[\s>]/);
	const locs = [...xml.matchAll(/<loc>\s*([^<]*?)\s*<\/loc>/g)].map((m) => m[1]);
	expect(locs).toEqual(['https://ordo.ilyasturki.com/']);
});

for (const file of ['robots.txt', 'sitemap.xml']) {
	test(`the private build has no ${file}`, async ({ request, build }) => {
		test.skip(build !== 'private', 'the site build has them');
		const res = await request.get('/' + file);
		expect(res.headers()['content-type']).toMatch(/^text\/html/);
		expect(await res.text()).not.toMatch(/sitemap|urlset|user-agent/i);
	});
}
