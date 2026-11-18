import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	define: {
		'import.meta.env.ORDO_WEB': JSON.stringify(process.env.ORDO_WEB ?? 'site')
	},
	server: {
		proxy: {
			'/api': process.env.ORDO_SERVE ?? 'http://127.0.0.1:7319'
		}
	},
	plugins: [tailwindcss(), sveltekit()]
});
