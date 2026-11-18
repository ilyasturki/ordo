import { expect, shoot, test } from './fixtures';

const repo = 'https://github.com/ilyasturki/ordo';

test.beforeEach(async ({ build }) => {
	test.skip(build !== 'site', 'the landing is the site build’s');
});

test('the landing shows the three surfaces, light and dark, without loading the core', async ({ page }, info) => {
	const wasm: string[] = [];
	page.on('request', (request) => {
		if (request.url().endsWith('.wasm')) wasm.push(request.url());
	});
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Plan your projects one version at a time');
	await expect(page.getByRole('img', { name: 'The same project in the terminal, the browser and the Android app' })).toBeVisible();

	const phone = page.locator('.s-phone img');
	if (info.project.name.endsWith('desktop')) {
		await phone.scrollIntoViewIfNeeded();
		await expect.poll(() => phone.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth)).toBe(480);
	} else {
		await expect(phone).toBeHidden();
	}

	const overflow = () => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
	expect(await overflow()).toBe(0);
	if (info.project.name.endsWith('phone')) {
		await page.setViewportSize({ width: 360, height: 780 });
		expect(await overflow()).toBe(0);
		await page.setViewportSize({ width: 390, height: 844 });
	}

	await shoot(page, 'landing', info.project.name);
	expect(wasm).toEqual([]);
});

test('the landing holds still while its fonts and images load', async ({ page, browserName }) => {
	test.skip(browserName !== 'chromium', 'only Chromium reports layout shifts');
	await page.addInitScript(() => {
		(window as unknown as { shift: number }).shift = 0;
		new PerformanceObserver((list) => {
			for (const entry of list.getEntries() as unknown as { value: number; hadRecentInput: boolean }[]) {
				if (!entry.hadRecentInput) (window as unknown as { shift: number }).shift += entry.value;
			}
		}).observe({ type: 'layout-shift', buffered: true });
	});
	await page.goto('/');
	await page.evaluate(() => document.fonts.ready);
	await page.waitForLoadState('load');
	expect(await page.evaluate(() => (window as unknown as { shift: number }).shift)).toBeLessThan(0.02);
});

test('the landing carries its title, description, canonical and Open Graph card', async ({ page, request }) => {
	await page.goto('/');
	await expect(page).toHaveTitle('Ordo: plan your projects one version at a time');
	const meta = (selector: string) => page.locator(selector).getAttribute('content');
	expect(await meta('meta[name="description"]')).toContain('keyboard-driven project planner');
	expect(await page.locator('link[rel="canonical"]').getAttribute('href')).toBe('https://ordo.ilyasturki.com/');
	expect(await meta('meta[property="og:url"]')).toBe('https://ordo.ilyasturki.com/');
	const image = await meta('meta[property="og:image"]');
	expect(image).toBe('https://ordo.ilyasturki.com/landing/social-preview.png');

	const card = await request.get(new URL(image!).pathname);
	expect(card.status()).toBe(200);
	expect(card.headers()['content-type']).toBe('image/png');
	for (const scheme of ['light', 'dark']) {
		expect((await request.get(`/landing/phone-${scheme}.webp`)).status()).toBe(200);
	}
	expect(await page.context().cookies()).toEqual([]);
});

test('every link on the landing answers', async ({ page, app }) => {
	await page.goto('/');
	const hrefs = await page.locator('a[href]').evaluateAll((links) => links.map((a) => a.getAttribute('href')!));
	const external = [...new Set(hrefs.filter((href) => href.startsWith('http')))].sort();
	expect(external).toEqual(
		[repo, `${repo}#install`, `${repo}/blob/main/CHANGELOG.md`, `${repo}/blob/main/docs/file-format.md`, `${repo}/releases/latest`].sort()
	);
	for (const href of hrefs.filter((href) => href.startsWith('#'))) {
		await expect(page.locator(href)).toHaveCount(1);
	}

	await page.getByRole('link', { name: 'Install', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Get Ordo' })).toBeInViewport();

	await page.locator('#get').getByRole('link', { name: 'Open Ordo in your browser' }).click();
	await expect(page).toHaveURL(app());
	await expect(page.getByRole('heading', { name: 'Projects' })).toBeVisible();

	await page.goBack();
	await page.getByRole('link', { name: 'Ordo', exact: true }).click();
	await expect(page).toHaveURL('/');
	await page.getByRole('main').getByRole('link', { name: 'Open Ordo in your browser' }).first().click();
	await expect(page.getByRole('heading', { name: 'Projects' })).toBeVisible();
});

test('the version card bumps by its buttons and by the TUI keys', async ({ page }) => {
	await page.goto('/');
	const version = page.getByTestId('bump-version');
	const card = page.getByRole('application', { name: /Version bump demo/ });

	await card.getByRole('button', { name: 'ctrl+a patch' }).click();
	await expect(version).toHaveText('v1.3.1');
	await card.getByRole('button', { name: 'gm minor' }).click();
	await expect(version).toHaveText('v1.4.0');

	await card.focus();
	await page.keyboard.press('g');
	await page.keyboard.press('Shift+M');
	await expect(version).toHaveText('v2.0.0');
	await page.keyboard.press('Control+a');
	await expect(version).toHaveText('v2.0.1');
	await page.keyboard.press('Control+x');
	await page.keyboard.press('Control+x');
	await expect(version).toHaveText('v2.0.0');
	await expect(card.getByRole('status')).toHaveText('the version is already at 0');
	await page.keyboard.press('m');
	await expect(version).toHaveText('v2.0.0');
});

test('a copy button copies its install command, or selects it where the clipboard is shut', async ({ page }) => {
	await page.goto('/#get');
	const button = page.locator('.cmd', { hasText: 'nix run' }).getByRole('button');
	await expect(button).toHaveAccessibleName('Copy the Nix command');
	await button.click();
	await expect
		.poll(async () =>
			(await button.getAttribute('aria-label')) === 'Copied' ? 'copied' : await page.evaluate(() => getSelection()?.toString())
		)
		.toMatch(/^(copied|nix run github:ilyasturki\/ordo)$/);
});
