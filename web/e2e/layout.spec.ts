import { expect, test } from './fixtures';
import { fitsWidth, pane, phone, taskLink, withFerry } from './ordo';

test('nothing scrolls sideways at 360 px, on the list, the task and the sheets', async ({ page, app }, info) => {
	test.skip(!phone(info), 'the narrowest phone');
	await page.setViewportSize({ width: 360, height: 740 });
	await withFerry(page, app());
	await fitsWidth(page);
	await page.getByRole('button', { name: 'Filter', exact: true }).click();
	await fitsWidth(page);
	await page.keyboard.press('Escape');
	await page.getByRole('button', { name: 'Add a task' }).click();
	await fitsWidth(page);
	await page.keyboard.press('Escape');
	await taskLink(page, 'Per-host overrides').click();
	await expect(pane(page)).toBeVisible();
	await fitsWidth(page);
	await page.getByRole('button', { name: 'More options' }).or(pane(page).getByRole('button', { name: 'More' })).first().click();
	await fitsWidth(page);
});

test('a wide screen lays the projects, the list and the task side by side', async ({ page, app }, info) => {
	test.skip(phone(info), 'the wide screen’s');
	await withFerry(page, app());
	await taskLink(page, 'Per-host overrides').click();
	const nav = await page.getByRole('navigation', { name: 'Projects' }).boundingBox();
	const list = await page.getByRole('region', { name: 'Tasks' }).boundingBox();
	const task = await pane(page).boundingBox();
	expect(nav!.x + nav!.width).toBeLessThanOrEqual(list!.x + 1);
	expect(list!.x + list!.width).toBeLessThanOrEqual(task!.x + 1);
	// The list scrolls on its own, and the bar stays.
	await page.getByRole('region', { name: 'Tasks' }).evaluate((el) => el.scrollTo(0, 10_000));
	await expect(page.getByRole('button', { name: 'Search' })).toBeInViewport();
	await expect(pane(page).getByRole('textbox', { name: 'Title' })).toBeInViewport();
});

test('the bar names the category in sight, where the + adds', async ({ page, app }) => {
	await withFerry(page, app());
	const bar = page.getByRole('heading', { level: 1 });
	await expect(bar).toContainText('v1.3.0');
	await page.getByRole('button', { name: 'v1.2.0, category menu' }).scrollIntoViewIfNeeded();
	await page.getByRole('button', { name: 'FUTURE, category menu' }).scrollIntoViewIfNeeded();
	await expect(bar).toContainText('FUTURE');
	await page.getByRole('button', { name: 'Add a task' }).click();
	await expect(page.getByRole('dialog', { name: 'New task' }).getByRole('button', { name: 'Category: FUTURE' })).toBeVisible();
});
