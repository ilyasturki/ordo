import adapter from '@sveltejs/adapter-static';

const build = process.env.ORDO_WEB ?? 'site';
if (build !== 'site' && build !== 'private') {
	throw new Error(`ORDO_WEB is site or private, not ${build}`);
}

/** @type {import('@sveltejs/kit').Config} */
export default {
	kit: {
		adapter: adapter({ pages: `build/${build}`, assets: `build/${build}`, fallback: '200.html' }),
		outDir: process.env.ORDO_WEB ? `.svelte-kit-${build}` : '.svelte-kit',
		prerender: { entries: build === 'site' ? ['/'] : [], handleUnseenRoutes: 'ignore' }
	}
};
