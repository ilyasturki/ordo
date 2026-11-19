import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const build = process.env.ORDO_WEB ?? 'site';
if (build !== 'site' && build !== 'private') {
	throw new Error(`ORDO_WEB is site or private, not ${build}`);
}

export default defineConfig({
	define: {
		'import.meta.env.ORDO_WEB': JSON.stringify(build)
	},
	server: {
		proxy: {
			'/api': process.env.ORDO_SERVE ?? 'http://127.0.0.1:7319'
		}
	},
	preview: {
		headers: process.env.ORDO_CSP ? { 'Content-Security-Policy': process.env.ORDO_CSP } : {}
	},
	plugins: [
		tailwindcss(),
		sveltekit({
			adapter: adapter({ pages: `build/${build}`, assets: `build/${build}`, fallback: '200.html' }),
			outDir: process.env.ORDO_WEB ? `.svelte-kit-${build}` : '.svelte-kit',
			csp: { mode: 'hash', directives: { 'script-src': ['self', 'wasm-unsafe-eval'] } },
			prerender: { entries: build === 'site' ? ['/', '/robots.txt', '/sitemap.xml'] : [], handleUnseenRoutes: 'ignore' },
			version: { pollInterval: 0 }
		})
	]
});
