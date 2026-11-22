import type { Page } from '@playwright/test';
import { expect, shoot, test } from './fixtures';
import { box, drag, filter, list, middle, row, section, titles, withDrag } from './drag';
import { phone } from './ordo';

const inbox = ['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Call the bank', 'Renew passport'];
const done = 'dddddddd-0000-4000-8000-000000000012';

/** The list item that holds Done's line, or its separator once it shows. */
const doneLine = (page: Page) => list(page).locator(`[data-line="t:${done}"]`);

const cursor = (page: Page) => page.locator('[data-key].bg-secondary-container');

const inboxHidden = (page: Page) => list(page).locator('[data-key="h:cccccccc-0000-4000-8000-000000000018"]');

test('under a filter a section it hides whole shows as one line in place, which a click reveals and hides', async ({ page, app }, info) => {
	await withDrag(page, app());
	await filter(page, 'Todo');
	const line = section(page, 'Done · 2 hidden');
	await expect(line).toHaveAttribute('aria-expanded', 'false');
	await expect(row(page, 'Book the ferry')).toHaveCount(0);
	await expect(section(page, 'Later · 2 hidden')).toBeVisible();
	await expect(section(page, 'Head · 1 hidden')).toBeVisible();
	await expect(list(page).locator('[data-key^="h:"]').filter({ hasText: '2 hidden by the filter' }), 'the runs aren’t counted again').toHaveCount(0);
	const above = await box(row(page, 'Plan the trip'));
	expect((await box(line)).y, 'in its place, under the row above').toBeGreaterThanOrEqual(above.y + above.height - 1);
	expect((await box(line)).y + (await box(line)).height).toBeLessThanOrEqual((await box(row(page, 'Next'))).y + 1);
	await shoot(page, 'hidden-section', info.project.name);

	await line.click();
	const shown = section(page, 'Done · hide 2');
	await expect(shown).toHaveAttribute('aria-expanded', 'true');
	await expect(row(page, 'Book the ferry')).toBeVisible();
	await expect(row(page, 'Pack the bags')).toBeVisible();
	await expect(row(page, 'Book the ferry')).toHaveClass(/opacity-50/);
	await shoot(page, 'hidden-section-revealed', info.project.name);
	await shown.click();
	await expect(row(page, 'Book the ferry')).toHaveCount(0);
	await expect(line).toBeVisible();
	await expect(page).toHaveURL(app('/p/0rd0-web-fixture'));
});

for (const [name, chips] of [
	['a status filter', ['Todo']],
	['a tag filter', ['Green']]
] as const) {
	test(`under ${name} a row dropped between a section’s line and the next row ends that section, and one dropped above the line goes before it`, async ({
		page,
		app
	}, info) => {
		await withDrag(page, app());
		await filter(page, ...chips);
		await drag(page, info, 'Call the bank', middle(await box(row(page, 'Next'))) - middle(await box(row(page, 'Call the bank'))));
		await expect.poll(() => titles(page, 'Inbox')).toEqual(['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Call the bank', 'Next', 'Renew passport']);
		await expect(row(page, 'Done'), 'its separator shows with the row').toBeVisible();
		await expect(row(page, 'Book the ferry')).toHaveCount(0);
		await expect(inboxHidden(page), 'its other rows join the count').toHaveText('2 hidden by the filter');
		await shoot(page, 'hidden-section-dropped', info.project.name);

		await drag(page, info, 'Call the bank', middle(await box(row(page, 'Done'))) - middle(await box(row(page, 'Call the bank'))));
		await expect.poll(() => titles(page, 'Inbox')).toEqual(['Plan the trip', 'Call the bank', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Renew passport']);
		await expect(section(page, 'Done · 2 hidden'), 'collapsed again').toBeVisible();
		await expect(inboxHidden(page)).toHaveCount(0);
	});
}

test('under a filter a section’s line drags its separator alone', async ({ page, app }, info) => {
	await withDrag(page, app());
	await filter(page, 'Todo');
	await drag(page, info, doneLine(page), middle(await box(row(page, 'Plan the trip'))) - middle(await box(doneLine(page))));
	await expect.poll(() => titles(page, 'Inbox')).toEqual(['Done', 'Plan the trip', 'Book the ferry', 'Pack the bags', 'Next', 'Call the bank', 'Renew passport']);
	await expect(row(page, 'Done'), 'a separator over the row it took').toBeVisible();
	await expect(row(page, 'Book the ferry'), 'the drop doesn’t reveal its rows').toHaveCount(0);
});

test('the keys stop on a section’s line, enter and space reveal it, and K steps over it as one row', async ({ page, app }, info) => {
	test.skip(phone(info), 'the keys are the wide screen’s');
	await withDrag(page, app());
	await filter(page, 'Todo');
	// The sheet hands focus back to its button, which would take enter.
	await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
	for (const key of ['j', 'j', 'j']) await page.keyboard.press(key);
	await expect(cursor(page)).toHaveAttribute('data-key', `t:${done}`);
	await shoot(page, 'hidden-section-cursor', info.project.name);
	await page.keyboard.press('Enter');
	await expect(section(page, 'Done · hide 2')).toBeVisible();
	await expect(page.getByRole('article', { name: 'Task' }), 'enter reveals, doesn’t open').toBeHidden();
	await page.keyboard.press(' ');
	await expect(section(page, 'Done · 2 hidden')).toBeVisible();

	await page.keyboard.press('j');
	await page.keyboard.press('j');
	await expect(cursor(page)).toContainText('Call the bank');
	await page.keyboard.press('K');
	await expect.poll(() => titles(page, 'Inbox')).toEqual(['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Call the bank', 'Next', 'Renew passport']);
	await expect(cursor(page)).toContainText('Call the bank');
	await page.keyboard.press('K');
	await expect.poll(() => titles(page, 'Inbox')).toEqual(['Plan the trip', 'Call the bank', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Renew passport']);
	await expect(section(page, 'Done · 2 hidden')).toBeVisible();
	await page.keyboard.press('u');
	await page.keyboard.press('u');
	await expect.poll(() => titles(page, 'Inbox')).toEqual(inbox);

	await page.keyboard.press('k');
	await page.keyboard.press('k');
	await expect(cursor(page)).toHaveAttribute('data-key', `t:${done}`);
	await page.keyboard.press('K');
	await expect.poll(() => titles(page, 'Inbox'), 'the line moves its separator alone').toEqual(['Done', 'Plan the trip', 'Book the ferry', 'Pack the bags', 'Next', 'Call the bank', 'Renew passport']);
});
