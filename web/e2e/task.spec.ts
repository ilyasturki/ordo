import { expect, shoot, test } from './fixtures';
import { pane, phone, row, taskLink, withFerry } from './ordo';

// A 1×1 png: the bytes name it, whatever the file is called.
const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==', 'base64');

test('a task opens as a page on a phone and beside the list on a wide screen', async ({ page, app }, info) => {
	await withFerry(page, app());
	await taskLink(page, 'Encrypt secrets with age').click();
	await expect(pane(page).getByRole('textbox', { name: 'Title' })).toHaveValue('Encrypt secrets with age');
	await expect(page).toHaveURL(new RegExp(`${app('/p')}/[^/]+/t/[^/]+$`));
	const list = page.getByRole('list', { name: 'ferry' });
	if (phone(info)) await expect(list).toBeHidden();
	else await expect(list).toBeVisible();
	await expect(pane(page).getByRole('list', { name: 'Subtasks' }).getByRole('link')).toHaveText([
		'Generate a key on first run',
		'Decrypt on apply',
		'Rotate keys without re-cloning'
	]);
	await shoot(page, 'task', info.project.name);

	await pane(page).getByRole('link', { name: 'Decrypt on apply' }).click();
	await expect(pane(page).getByRole('link', { name: 'Subtask of Encrypt secrets with age' })).toBeVisible();

	if (phone(info)) {
		await page.goBack();
		await page.goBack();
		await expect(list).toBeVisible();
	} else {
		await pane(page).getByRole('button', { name: 'Close' }).click();
		await expect(pane(page)).toBeHidden();
	}
});

test('the task page edits title, description, status, priority, colour and subtasks', async ({ page, app }) => {
	await withFerry(page, app());
	await taskLink(page, 'Dry-run flag').click();
	const title = pane(page).getByRole('textbox', { name: 'Title' });
	await title.fill('Dry-run flag for apply and sync');
	// Esc keeps what was typed.
	await title.press('Escape');
	await expect(title).not.toBeFocused();
	await pane(page).getByRole('textbox', { name: 'Description' }).fill('Prints what would change.');
	await pane(page).getByRole('textbox', { name: 'Description' }).blur();

	await pane(page).getByRole('button', { name: 'Priority: Low' }).click();
	await page.getByRole('menuitemradio', { name: 'Critical' }).click();
	await expect(pane(page).getByRole('button', { name: 'Priority: Critical' })).toBeVisible();
	await pane(page).getByRole('button', { name: 'Colour', exact: true }).click();
	await page.getByRole('menuitemradio', { name: 'Cyan' }).click();
	await expect(pane(page).getByRole('button', { name: 'Colour: Cyan' })).toBeVisible();
	await pane(page).getByRole('button', { name: 'Status: Todo' }).click();
	await page.getByRole('menuitemradio', { name: 'In Progress' }).click();
	await expect(pane(page).getByRole('button', { name: 'Status: In Progress' })).toBeVisible();

	await pane(page).getByRole('textbox', { name: 'Add a subtask' }).fill('Show a diff');
	await page.keyboard.press('Enter');
	await expect(pane(page).getByRole('list', { name: 'Subtasks' }).getByRole('link', { name: 'Show a diff' })).toBeVisible();

	await page.reload();
	await expect(title).toHaveValue('Dry-run flag for apply and sync');
	await expect(pane(page).getByRole('textbox', { name: 'Description' })).toHaveValue('Prints what would change.');
	await expect(pane(page).getByRole('button', { name: 'Priority: Critical' })).toBeVisible();

	await pane(page).getByRole('button', { name: 'Mark done' }).click();
	await expect(pane(page).getByRole('button', { name: 'Mark todo' })).toBeVisible();
});

test('a task moves to another category, up and down, and deletes with an Undo', async ({ page, app }, info) => {
	await withFerry(page, app());
	await taskLink(page, 'Page long output').click();
	await pane(page).getByRole('button', { name: 'In next, move to another category' }).click();
	await page.getByRole('menuitem', { name: 'v1.3.0', exact: true }).click();
	await expect(pane(page).getByRole('button', { name: 'In v1.3.0, move to another category' })).toBeVisible();

	const list = page.getByRole('list', { name: 'ferry' });
	await pane(page).getByRole('button', { name: 'More' }).click();
	await page.getByRole('menuitem', { name: 'Move up' }).click();
	if (!phone(info)) {
		await expect.poll(async () => {
			const titles = await list.getByRole('link').allInnerTexts();
			return titles.findIndex((t) => t.startsWith('Page long output')) - titles.findIndex((t) => t.startsWith('Dry-run flag'));
		}).toBe(-1);
	}
	const before = phone(info) ? [] : await list.getByRole('link').allInnerTexts();
	await pane(page).getByRole('button', { name: 'More' }).click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	if (phone(info)) await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();
	await expect(taskLink(page, 'Page long output')).toBeHidden();
	await page.getByRole('status').getByRole('button', { name: 'Undo' }).click();
	await expect(taskLink(page, 'Page long output')).toBeVisible();
	if (!phone(info)) expect(await list.getByRole('link').allInnerTexts()).toEqual(before);
});

test('an image attaches from a file, opens full size and goes again', async ({ page, app }, info) => {
	await withFerry(page, app());
	await taskLink(page, 'Shell completions').click();
	const chooser = page.waitForEvent('filechooser');
	await pane(page).getByRole('button', { name: 'Add an image' }).click();
	await (await chooser).setFiles({ name: 'fish prompt.png', mimeType: 'image/png', buffer: png });
	await expect(pane(page).getByRole('img', { name: 'fish prompt' })).toBeVisible();
	await shoot(page, 'image', info.project.name);
	await pane(page).getByRole('button', { name: 'Open fish prompt' }).click();
	await expect(page.getByRole('dialog', { name: 'fish prompt' })).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(page.getByRole('dialog', { name: 'fish prompt' })).toBeHidden();

	if (phone(info)) await pane(page).getByRole('button', { name: 'Back' }).click();
	await expect(row(page, 'Shell completions').getByText('1', { exact: true })).toBeVisible();
	if (phone(info)) await taskLink(page, 'Shell completions').click();

	await pane(page).getByRole('button', { name: 'Remove fish prompt' }).click();
	await expect(pane(page).getByRole('img', { name: 'fish prompt' })).toBeHidden();
	await expect(page.getByRole('status')).toHaveText(/Image removed/);
});
