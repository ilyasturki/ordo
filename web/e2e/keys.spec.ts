import type { Page } from '@playwright/test';
import { expect, shoot, test } from './fixtures';
import { pane, phone, row, taskLink, withFerry } from './ordo';

/** Waits for the sheet a key opened to close, as the next key goes to the list only then. */
const closed = (page: Page) => expect(page.locator('dialog[open]')).toHaveCount(0);

const cursor = (page: Page) => page.locator('[data-key].bg-secondary-container');

const titles = (page: Page) => page.getByRole('list', { name: 'ferry' }).getByRole('link').allInnerTexts();

test.beforeEach(({}, info) => {
	test.skip(phone(info), 'the keys are the wide screen’s');
});

test('j and k walk the rows, enter opens the task beside them and esc comes back', async ({ page, app }, info) => {
	await withFerry(page, app());
	await page.keyboard.press('j');
	await expect(cursor(page)).toHaveAttribute('data-key', /^c:/);
	await page.keyboard.press('j');
	await expect(cursor(page)).toContainText('Shell completions for fish');
	await page.keyboard.press('ArrowDown');
	await expect(cursor(page)).toContainText('Page long output');
	await page.keyboard.press('k');
	await page.keyboard.press('Enter');
	const title = pane(page).getByRole('textbox', { name: 'Title' });
	await expect(title).toBeFocused();
	await expect(title).toHaveValue('Shell completions for fish');
	await title.press('Escape');
	// The pane follows the cursor.
	await page.keyboard.press('j');
	await expect(title).toHaveValue('Page long output in ferry diff');
	await page.keyboard.press('Escape');
	await expect(pane(page)).toBeHidden();
	await page.keyboard.press('G');
	await expect(cursor(page)).toBeVisible();
	await page.keyboard.press('g');
	await page.keyboard.press('g');
	await expect(cursor(page)).toHaveAttribute('data-key', /^c:/);
	await page.keyboard.press('?');
	await expect(page.getByRole('dialog', { name: 'Keyboard shortcuts' })).toBeVisible();
	await shoot(page, 'keys', info.project.name);
	await page.keyboard.press('Escape');
});

test('a, o, - and A add below the cursor; x and space cycle; d deletes; u and ctrl+r undo and redo', async ({ page, app }) => {
	await withFerry(page, app());
	await page.keyboard.press('j');
	await page.keyboard.press('j');
	await page.keyboard.press('a');
	await page.getByRole('textbox', { name: 'New task' }).fill('Bash completions');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('textbox', { name: 'New task' })).toHaveValue('');
	await page.keyboard.press('Escape');
	await closed(page);
	expect((await titles(page)).slice(0, 2)).toEqual(['Shell completions for fish', 'Bash completions']);
	await expect(cursor(page)).toContainText('Bash completions');

	await page.keyboard.press('o');
	await page.getByRole('textbox', { name: 'Note' }).fill('zsh too?');
	await page.keyboard.press('Enter');
	await closed(page);
	await page.keyboard.press('-');
	await expect(page.getByRole('textbox', { name: 'Separator label' })).toBeFocused();
	await page.keyboard.press('Enter');
	await closed(page);
	await expect.poll(async () => (await titles(page)).slice(0, 4)).toEqual(['Shell completions for fish', 'Bash completions', 'zsh too?', '']);

	await page.keyboard.press('g');
	await page.keyboard.press('g');
	await page.keyboard.press('j');
	await page.keyboard.press('x');
	await expect(page.locator('[data-key].bg-secondary-container').getByRole('button', { name: 'In Progress, change status' })).toBeVisible();
	await page.keyboard.press(' ');
	await expect(cursor(page).getByRole('button', { name: 'Completed, change status' })).toBeVisible();
	await page.keyboard.press('u');
	await expect(cursor(page).getByRole('button', { name: 'In Progress, change status' })).toBeVisible();
	await page.keyboard.press('Control+r');
	await expect(cursor(page).getByRole('button', { name: 'Completed, change status' })).toBeVisible();

	await page.keyboard.press('d');
	await expect(taskLink(page, 'Shell completions')).toBeHidden();
	await page.keyboard.press('u');
	await expect(taskLink(page, 'Shell completions')).toBeVisible();

	await page.keyboard.press('A');
	await page.getByRole('textbox', { name: 'New category' }).fill('Ideas');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('button', { name: 'Ideas, category menu' })).toBeVisible();
});

test('J and K move, > and < nest, h and l change the priority, m moves to a category', async ({ page, app }) => {
	await withFerry(page, app());
	await page.keyboard.press('j');
	await page.keyboard.press('j');
	await page.keyboard.press('J');
	await expect.poll(async () => (await titles(page))[0]).toMatch(/^Page long output/);
	await page.keyboard.press('K');
	await expect.poll(async () => (await titles(page))[0]).toMatch(/^Shell completions/);

	await page.keyboard.press('j');
	await page.keyboard.press('>');
	await expect(row(page, 'Shell completions').getByText('0/1')).toBeVisible();
	await page.keyboard.press('<');
	await expect(page.getByText('0/1')).toBeHidden();

	await page.keyboard.press('l');
	await expect(cursor(page).getByText('Low')).toBeHidden();
	await page.keyboard.press('l');
	await expect(cursor(page).getByText('High')).toBeVisible();
	await page.keyboard.press('h');
	await page.keyboard.press('h');
	await expect(cursor(page).getByText('Low')).toBeVisible();

	await page.keyboard.press('m');
	await page.getByRole('dialog', { name: 'Move to' }).getByRole('button', { name: 'FUTURE' }).click();
	await closed(page);
	await expect(page.getByRole('button', { name: 'next, category menu' })).toBeVisible();
	await expect.poll(async () => (await titles(page))[0]).toMatch(/^Shell completions/);
});

test('/ searches and f filters from the keys', async ({ page, app }) => {
	await withFerry(page, app());
	await page.keyboard.press('/');
	await expect(page.getByRole('searchbox', { name: 'Search' })).toBeFocused();
	await page.keyboard.type('fish');
	await expect(taskLink(page, 'Encrypt secrets')).toBeHidden();
	await page.keyboard.press('Escape');
	await page.keyboard.press('f');
	await expect(page.getByRole('dialog', { name: 'Filter' })).toBeVisible();
	await page.keyboard.press('Escape');
});
