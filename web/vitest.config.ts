import { fileURLToPath } from 'node:url';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [svelte({ configFile: false })],
	resolve: {
		alias: { $lib: fileURLToPath(new URL('src/lib', import.meta.url)) }
	},
	define: {
		'import.meta.env.ORDO_WEB': '"site"'
	},
	test: {
		include: ['src/**/*.test.ts'],
		// Its ESM imports name no extension, which Node refuses.
		server: { deps: { inline: ['@material/material-color-utilities'] } }
	}
});
