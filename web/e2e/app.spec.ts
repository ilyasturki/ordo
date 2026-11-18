import { readFileSync } from 'node:fs';
import { expect, shoot, test } from './fixtures';

const ferry = new URL('../../assets/demo/ferry.json', import.meta.url);

test('a project made in the browser keeps its rows through a reload', async ({ page, app }, info) => {
	await page.goto(app());
	await expect(page.getByText('No projects yet.')).toBeVisible();
	await page.getByLabel('Project name').fill('Ferry');
	await page.getByRole('button', { name: 'Create' }).click();
	await expect(page).toHaveURL(new RegExp(`${app('/p')}/[0-9a-f-]+$`));
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Ferry');

	await page.getByLabel('Category name').fill('Fix');
	await page.getByRole('button', { name: 'Add' }).click();
	await page.getByLabel('Task title').fill('Dock at noon');
	await page.getByRole('button', { name: 'Add' }).click();
	await expect(page.getByRole('region', { name: 'Fix' }).getByText('Dock at noon')).toBeVisible();

	await page.reload();
	await expect(page.getByRole('region', { name: 'Fix' }).getByText('Dock at noon')).toBeVisible();
	await shoot(page, 'project', info.project.name);

	await page.getByRole('link', { name: 'Projects' }).click();
	await expect(page).toHaveURL(app());
	await expect(page.getByRole('link', { name: 'Ferry' })).toBeVisible();
	await page.getByLabel('Project name').fill('ferry');
	await page.getByRole('button', { name: 'Create' }).click();
	await expect(page.getByRole('alert')).toHaveText('A project with that name already exists');
	await shoot(page, 'projects', info.project.name);
});

test('a project file imports, and exports as ordo writes it', async ({ page, app }) => {
	await page.goto(app());
	await page.getByLabel('Import a project file').setInputFiles(ferry.pathname);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('ferry');
	await expect(page.getByRole('region', { name: 'next' })).toBeVisible();

	const download = page.waitForEvent('download');
	await page.getByRole('button', { name: 'Export' }).click();
	const file = await download;
	const id = JSON.parse(readFileSync(ferry, 'utf8')).id;
	expect(file.suggestedFilename()).toBe(`${id}.json`);
	const exported = JSON.parse(readFileSync(await file.path(), 'utf8'));
	expect(exported.schema).toBeGreaterThan(0);
	expect(exported.name).toBe('ferry');
});

test('a link to a project this browser lacks says so', async ({ page, app }) => {
	await page.goto(app('/p/nowhere'));
	await expect(page.getByText("This project isn't in this browser.")).toBeVisible();
});
