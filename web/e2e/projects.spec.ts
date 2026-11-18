import { readFileSync } from 'node:fs';
import { expect, shoot, test } from './fixtures';
import { ferry, phone, stored, withFerry } from './ordo';

test('a project is created, renamed, grouped and deleted in the browser', async ({ page, app }, info) => {
	await page.goto(app());
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	await shoot(page, 'empty', info.project.name);

	await page.getByRole('button', { name: 'New project' }).click();
	await page.getByRole('textbox', { name: 'New project' }).fill('Garden');
	await page.keyboard.press('Enter');
	await expect(page).toHaveURL(new RegExp(`${app('/p')}/[0-9a-f-]+$`));
	await expect(page.getByText('No categories yet')).toBeVisible();

	await page.getByRole('button', { name: 'New category' }).click();
	await page.getByRole('textbox', { name: 'New category' }).fill('Beds');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('button', { name: 'Beds, category menu' })).toBeVisible();

	await page.getByRole('button', { name: 'More options' }).click();
	await page.getByRole('menuitem', { name: 'Rename project' }).click();
	const name = page.getByRole('textbox', { name: 'Project name' });
	await name.fill('Allotment');
	// Esc keeps a name, as Enter does.
	await page.keyboard.press('Escape');
	await expect(page.getByRole('navigation', { name: 'Projects' }).getByRole('link', { name: 'Allotment' }).first()).toBeAttached();

	await page.getByRole('button', { name: 'More options' }).click();
	await page.getByRole('menuitem', { name: 'Group' }).click();
	await page.getByRole('textbox', { name: 'Group' }).fill('Home');
	await page.keyboard.press('Enter');

	await page.getByRole('button', { name: 'More options' }).click();
	await page.getByRole('menuitem', { name: 'New category' }).click();
	await page.getByRole('textbox', { name: 'New category' }).fill('beds');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('dialog', { name: '“Beds” is taken' })).toBeVisible();
	await page.getByRole('button', { name: 'Name it “beds (2)”' }).click();
	await expect(page.getByRole('button', { name: 'beds (2), category menu' })).toBeVisible();

	await page.reload();
	await expect(page.getByRole('button', { name: 'beds (2), category menu' })).toBeVisible();
	if (!phone(info)) await expect(page.getByRole('navigation', { name: 'Projects' }).getByText('Home')).toBeVisible();
	await shoot(page, 'project', info.project.name);

	await page.getByRole('button', { name: 'More options' }).click();
	await page.getByRole('menuitem', { name: 'Delete project' }).click();
	const confirm = page.getByRole('dialog', { name: 'Delete project “Allotment”?' });
	await expect(confirm).toContainText('Its 2 categories and 0 tasks go too.');
	await confirm.getByRole('button', { name: 'Delete' }).click();
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	await expect(page).toHaveURL(app());
});

test('projects switch from the tabs on a phone and the side list on a wide screen', async ({ page, app }, info) => {
	await withFerry(page, app());
	const nav = page.getByRole('navigation', { name: 'Projects' }).filter({ visible: true });
	await nav.getByRole('button', { name: 'New project' }).click();
	await page.getByRole('textbox', { name: 'New project' }).fill('Homelab');
	await page.keyboard.press('Enter');
	await expect(page.getByText('No categories yet')).toBeVisible();
	await expect(nav.getByRole('link', { name: 'Homelab' })).toHaveAttribute('aria-current', 'page');

	await nav.getByRole('link', { name: 'ferry' }).click();
	await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();
	await expect(nav.getByRole('link', { name: 'ferry' })).toHaveAttribute('aria-current', 'page');

	// A reload opens on the project last shown.
	const id = JSON.parse(readFileSync(ferry, 'utf8')).id;
	await expect.poll(() => stored(page, 'current')).toBe(id);
	await page.goto(app());
	await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();

	if (phone(info)) {
		// A tap on the open tab opens its sheet.
		await nav.getByRole('link', { name: 'ferry' }).click();
		await expect(page.getByRole('textbox', { name: 'Project name' })).toHaveValue('ferry');
		await page.keyboard.press('Escape');
	}
});

test('a project file imports, and exports as ordo writes it', async ({ page, app }) => {
	await withFerry(page, app());
	await expect(page.getByRole('status')).toHaveText('Imported ferry');

	await page.getByRole('button', { name: 'More options' }).click();
	const download = page.waitForEvent('download');
	await page.getByRole('menuitem', { name: 'Export project' }).click();
	const file = await download;
	const id = JSON.parse(readFileSync(ferry, 'utf8')).id;
	expect(file.suggestedFilename()).toBe(`${id}.json`);
	const exported = JSON.parse(readFileSync(await file.path(), 'utf8'));
	expect(exported.schema).toBeGreaterThan(0);
	expect(exported.name).toBe('ferry');

	// The same file again is refused: its name is taken.
	await page.getByRole('button', { name: 'More options' }).click();
	const chooser = page.waitForEvent('filechooser');
	await page.getByRole('menuitem', { name: 'Import a project' }).click();
	await (await chooser).setFiles(ferry);
	await expect(page.getByRole('status')).toHaveText(/Can't import ferry\.json: A project with that name already exists/);
});

test('a link to a project this browser lacks lands on the app', async ({ page, app }) => {
	await page.goto(app('/p/nowhere'));
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	await expect(page).toHaveURL(app());
});
