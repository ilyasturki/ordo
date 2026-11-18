import { expect, shoot, test } from './fixtures';
import { row, taskLink, withFerry } from './ordo';

test('search narrows the list to the rows that match, smartcase, and closes back to all', async ({ page, app }, info) => {
	await withFerry(page, app());
	await page.getByRole('button', { name: 'Search' }).click();
	const field = page.getByRole('searchbox', { name: 'Search' });
	await expect(field).toBeFocused();
	await field.fill('apply');
	await expect(taskLink(page, 'Decrypt on apply')).toBeVisible();
	await expect(taskLink(page, 'Dry-run flag for apply')).toBeVisible();
	await expect(taskLink(page, 'Shell completions')).toBeHidden();
	await shoot(page, 'search', info.project.name);
	await field.fill('Apply');
	await expect(page.getByText('Nothing matches')).toBeVisible();
	await field.press('Escape');
	await expect(field).toBeHidden();
	await expect(taskLink(page, 'Shell completions')).toBeVisible();
});

test('the filter hides rows, counts them, peeks at them, turns off and clears', async ({ page, app }, info) => {
	await withFerry(page, app());
	await page.getByRole('button', { name: 'Filter', exact: true }).click();
	const sheet = page.getByRole('dialog', { name: 'Filter' });
	await sheet.getByRole('button', { name: 'Todo' }).click();
	await expect(sheet.getByRole('button', { name: 'Todo' })).toHaveAttribute('aria-pressed', 'true');
	await shoot(page, 'filter', info.project.name);
	await page.keyboard.press('Escape');
	await expect(sheet).toBeHidden();

	await expect(page.getByRole('button', { name: 'Filter, on' })).toBeVisible();
	await expect(taskLink(page, 'Keep file modes on apply')).toBeHidden();
	const hidden = page.locator('[data-key="h:c-v121"]');
	await expect(hidden).toHaveText('2 hidden by the filter');
	await expect(hidden).toBeVisible();
	await hidden.click();
	await expect(taskLink(page, 'Keep file modes on apply')).toBeVisible();
	await expect(hidden).toHaveText('Hide the 2 filtered out');
	await hidden.click();
	await expect(taskLink(page, 'Keep file modes on apply')).toBeHidden();

	// A row an edit takes out of the filter stays where it was until the filter changes.
	await row(page, 'Shell completions').getByRole('button', { name: 'Todo, change status' }).click();
	await expect(taskLink(page, 'Shell completions')).toBeVisible();

	await page.getByRole('button', { name: 'Filter, on' }).click();
	await sheet.getByRole('switch', { name: 'Filter on' }).click();
	await page.keyboard.press('Escape');
	await expect(page.getByRole('button', { name: 'Filter, off' })).toBeVisible();
	await expect(taskLink(page, 'Keep file modes on apply')).toBeVisible();

	await page.reload();
	await expect(page.getByRole('button', { name: 'Filter, off' })).toBeVisible();
	await page.getByRole('button', { name: 'Filter, off' }).click();
	await sheet.getByRole('button', { name: 'Clear' }).click();
	await page.keyboard.press('Escape');
	await expect(page.getByRole('button', { name: 'Filter', exact: true })).toBeVisible();
});
