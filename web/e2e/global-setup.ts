import { chromium, firefox, webkit, type Browser, type FullConfig } from '@playwright/test';
import { ordo } from './serve';

const browsers = { chromium, firefox, webkit };

// WebKit without an EGL display launches fine and dies at its first page, its reason only on stderr.
async function opensAPage(engine: keyof typeof browsers): Promise<void> {
	let timer: NodeJS.Timeout | undefined;
	const timeout = new Promise<never>((_, reject) => {
		timer = setTimeout(() => reject(new Error('no page after 30 s')), 30_000);
	});
	let browser: Browser | undefined;
	try {
		browser = await Promise.race([browsers[engine].launch(), timeout]);
		const page = await Promise.race([browser.newPage(), timeout]);
		await Promise.race([page.setContent('<p>ordo</p>'), timeout]);
	} catch (error) {
		throw new Error(`${engine} cannot open a page (DEBUG=pw:browser shows its stderr): ${(error as Error).message}`);
	} finally {
		clearTimeout(timer);
		await browser?.close().catch(() => {});
	}
}

// Workers get the binary through the environment, so no test pays for the build in its own timeout.
export default async function setup(config: FullConfig): Promise<void> {
	process.env.ORDO_E2E_BIN = ordo();
	const engines = new Set(config.projects.map((project) => project.use.browserName ?? project.use.defaultBrowserType ?? 'chromium'));
	await Promise.all([...engines].map(opensAPage));
}
