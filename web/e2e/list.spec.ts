import { expect, shoot, test } from './fixtures';
import { row, taskLink, withFerry } from './ordo';

test('a status cycles from its icon, and done offers an Undo', async ({ page, app }) => {
	await withFerry(page, app());
	const shell = row(page, 'Shell completions');
	await shell.getByRole('button', { name: 'Todo, change status' }).click();
	await expect(shell.getByRole('button', { name: 'In Progress, change status' })).toBeVisible();
	await shell.getByRole('button', { name: 'In Progress, change status' }).click();
	await expect(shell.getByRole('button', { name: 'Completed, change status' })).toBeVisible();
	await page.getByRole('status').getByRole('button', { name: 'Undo' }).click();
	await expect(shell.getByRole('button', { name: 'In Progress, change status' })).toBeVisible();
});

test('categories and subtasks fold, and stay folded through a reload', async ({ page, app }) => {
	await withFerry(page, app());
	await page.getByRole('button', { name: 'Fold subtasks of Encrypt secrets with age' }).click();
	await expect(taskLink(page, 'Decrypt on apply')).toBeHidden();
	await page.getByRole('button', { name: 'Fold next' }).click();
	await expect(taskLink(page, 'Shell completions')).toBeHidden();
	await expect(page.getByRole('button', { name: 'Unfold next' })).toBeVisible();
	await page.reload();
	await expect(page.getByRole('button', { name: 'Unfold next' })).toBeVisible();
	await expect(taskLink(page, 'Decrypt on apply')).toBeHidden();
	await page.getByRole('button', { name: 'Unfold next' }).click();
	await expect(taskLink(page, 'Shell completions')).toBeVisible();
});

test('the add sheet adds a task with its priority and colour, and Esc keeps what was typed', async ({ page, app }, info) => {
	await withFerry(page, app());
	await page.getByRole('button', { name: 'Add a task' }).click();
	const sheet = page.getByRole('dialog', { name: 'New task' });
	await sheet.getByRole('textbox', { name: 'New task' }).fill('Sign the release');
	await sheet.getByRole('button', { name: 'Priority' }).click();
	await page.getByRole('menuitemradio', { name: 'High' }).click();
	await sheet.getByRole('button', { name: 'Colour' }).click();
	await page.getByRole('menuitemradio', { name: 'Green' }).click();
	await shoot(page, 'add', info.project.name);
	await page.keyboard.press('Enter');
	await expect(sheet.getByRole('textbox', { name: 'New task' })).toHaveValue('');
	await expect(row(page, 'Sign the release').getByText('High')).toBeVisible();

	await sheet.getByRole('textbox', { name: 'New task' }).fill('Tag the commit');
	await page.keyboard.press('Escape');
	await expect(sheet).toBeHidden();
	await expect(taskLink(page, 'Tag the commit')).toBeVisible();
	// Each lands under the one before, chained as the phone adds them.
	const titles = await page.getByRole('list', { name: 'ferry' }).getByRole('link').allInnerTexts();
	const at = titles.findIndex((t) => t.startsWith('Sign the release'));
	expect(titles[at + 1]).toMatch(/^Tag the commit/);
});

test('a category menu adds notes, separators and sub-categories, pins and deletes with an Undo', async ({ page, app }, info) => {
	await withFerry(page, app());
	await page.getByRole('button', { name: 'next, category menu' }).click();
	await page.getByRole('button', { name: 'Add a note' }).click();
	await page.getByRole('textbox', { name: 'Note' }).fill('Ask on the forum first');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('list', { name: 'ferry' }).getByText('Ask on the forum first')).toBeVisible();

	await page.getByRole('button', { name: 'next, category menu' }).click();
	await page.getByRole('button', { name: 'Add a separator' }).click();
	await page.getByRole('textbox', { name: 'Separator label' }).fill('Later');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('link', { name: 'Later' })).toBeVisible();

	await page.getByRole('button', { name: 'next, category menu' }).click();
	await page.getByRole('button', { name: 'Pin' }).click();
	await expect(page.getByRole('button', { name: 'next, category menu' }).locator('svg')).toHaveCount(1);

	await page.getByRole('button', { name: 'v1.3.0, category menu' }).click();
	await page.getByRole('button', { name: 'New sub-category' }).click();
	await page.getByRole('textbox', { name: 'New sub-category' }).fill('tests');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('button', { name: 'tests, category menu' })).toBeVisible();
	await shoot(page, 'menu-adds', info.project.name);

	await page.getByRole('button', { name: 'v1.2.1, category menu' }).click();
	await page.getByRole('button', { name: 'Add next version' }).click();
	await expect(page.getByRole('button', { name: 'v1.4.0, category menu' })).toBeVisible();

	await page.getByRole('button', { name: 'next, category menu' }).click();
	await page.getByRole('button', { name: 'Delete category' }).click();
	const confirm = page.getByRole('dialog', { name: 'Delete category “next”?' });
	await expect(confirm).toContainText('Its 2 tasks go too.');
	await confirm.getByRole('button', { name: 'Delete' }).click();
	await expect(page.getByRole('button', { name: 'next, category menu' })).toBeHidden();
	await page.getByRole('status').getByRole('button', { name: 'Undo' }).click();
	await expect(taskLink(page, 'Shell completions')).toBeVisible();
});
