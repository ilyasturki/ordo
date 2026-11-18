import { createServer } from 'node:net';
import { defineConfig, devices, type PlaywrightTestOptions, type Project } from '@playwright/test';
import type { Options } from './e2e/fixtures';
import { csp } from './e2e/front';

function freePort(): Promise<number> {
	return new Promise((resolve, reject) => {
		const server = createServer();
		server.on('error', reject);
		server.listen(0, '127.0.0.1', () => {
			const { port } = server.address() as { port: number };
			server.close(() => resolve(port));
		});
	});
}

// Workers load this file again: the runner picks the ports once and hands them down in the environment.
process.env.ORDO_E2E_SITE ??= String(await freePort());

// The private build runs behind a front of each test's own, with its ordo serve: see e2e/front.ts.
const builds = { site: Number(process.env.ORDO_E2E_SITE), private: 0 };
const engines = { chromium: devices['Desktop Chrome'], firefox: devices['Desktop Firefox'], webkit: devices['Desktop Safari'] };
const viewports = { phone: { width: 390, height: 844 }, desktop: { width: 1440, height: 900 } };

const projects: Project<PlaywrightTestOptions & Options>[] = [];
for (const [build, port] of Object.entries(builds)) {
	for (const [engine, device] of Object.entries(engines)) {
		for (const [size, viewport] of Object.entries(viewports)) {
			projects.push({
				name: `${build}-${engine}-${size}`,
				use: { ...device, viewport, baseURL: `http://127.0.0.1:${port}`, build: build as Options['build'] }
			});
		}
	}
}

export default defineConfig<Options>({
	testDir: 'e2e',
	globalSetup: './e2e/global-setup.ts',
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? 'github' : 'list',
	projects,
	webServer: {
		command: `bunx vite preview --host 127.0.0.1 --port ${builds.site} --strictPort`,
		env: { ORDO_WEB: 'site', ORDO_CSP: csp('http://127.0.0.1:*') },
		url: `http://127.0.0.1:${builds.site}`,
		reuseExistingServer: false
	}
});
