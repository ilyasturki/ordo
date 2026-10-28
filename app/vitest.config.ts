import { fileURLToPath } from 'node:url';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [svelte({ configFile: false, compilerOptions: { runes: true } })],
	resolve: {
		alias: { $lib: fileURLToPath(new URL('src/lib', import.meta.url)) }
	},
	define: {
		'import.meta.env.APP_BUILD': 'false'
	},
	test: {
		include: ['src/**/*.test.ts']
	}
});
