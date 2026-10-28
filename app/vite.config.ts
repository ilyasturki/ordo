import staticAdapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const isAppBuild = process.env.BUILD_TARGET === 'app';

export default defineConfig({
	define: {
		'import.meta.env.APP_BUILD': JSON.stringify(isAppBuild)
	},
	server: {
		proxy: {
			'/api': process.env.ORDO_SERVE ?? 'http://127.0.0.1:7319'
		}
	},
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/u).includes('node_modules') ? undefined : true
			},
			adapter: staticAdapter({ fallback: 'index.html' })
		})
	]
});
