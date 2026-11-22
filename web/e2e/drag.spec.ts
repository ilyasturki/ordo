import { expect, shoot, test } from './fixtures';
import { box, centre, drag, file, filter, hand, header, list, middle, open, pid, row, section, titles, withDrag } from './drag';
import { phone } from './ordo';

test('a row dropped above a separator with no filter lands there', async ({ page, app }, info) => {
	await withDrag(page, app());
	await drag(page, info, 'Call the bank', middle(await box(row(page, 'Next'))) - middle(await box(row(page, 'Call the bank'))));
	await expect
		.poll(() => titles(page, 'Inbox'))
		.toEqual(['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Call the bank', 'Next', 'Renew passport']);
});

test('under a filter a row moved last in a category leaves its hidden tail hidden', async ({ page, app }, info) => {
	await withDrag(page, app());
	await filter(page, ...open);
	await drag(page, info, 'Write the report', middle(await box(row(page, 'Review the slides'))) - middle(await box(row(page, 'Write the report'))));
	await expect.poll(() => titles(page, 'Work')).toEqual(['Review the slides', 'Write the report', 'Later', 'Old ticket', 'Older ticket']);
	await expect(section(page, 'Later · 2 hidden'), 'the tail stays behind its line').toBeVisible();
});

for (const [name, chips] of [
	['', []],
	['under a filter ', open]
] as const) {
	test(`${name}a row dragged up across a header lands last above it`, async ({ page, app }, info) => {
		await withDrag(page, app());
		if (chips.length) await filter(page, ...chips);
		const at = await box(row(page, 'Write the report'));
		await drag(page, info, 'Write the report', middle(await box(header(page, 'Work'))) - at.y - at.height * 0.15, undefined, () =>
			chips.length ? shoot(page, 'drag-lifted', info.project.name) : Promise.resolve()
		);
		await expect
			.poll(() => titles(page, 'Inbox'))
			.toEqual(['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Call the bank', 'Renew passport', 'Write the report']);
		const under = await row(page, 'Write the report').evaluate((el) => el.closest('[role="listitem"]')!.nextElementSibling?.getAttribute('data-line'));
		expect(under, 'last of Inbox, right above the Work header').toBe('c:cccccccc-0000-4000-8000-000000000024');
		if (chips.length) await shoot(page, 'drag-across-header', info.project.name);
	});
}

for (const [name, chips] of [
	['', []],
	['under a filter ', open]
] as const) {
	test(`${name}a row dragged just into an empty category above stays there`, async ({ page, app }, info) => {
		await withDrag(page, app());
		if (chips.length) await filter(page, ...chips);
		await centre(header(page, 'Home'));
		const at = await box(row(page, 'Fix the sink'));
		await drag(page, info, 'Fix the sink', middle(await box(header(page, 'Home'))) - at.y - at.height * 0.15, 800);
		await expect.poll(() => titles(page, 'Empty')).toEqual(['Fix the sink']);
		expect(await titles(page, 'Home')).toEqual(['Paint the door', 'Mow the lawn', 'Clean the gutter']);
	});
}

test('under a filter a row dragged up through an empty category leaves its no tasks line and lands at the end of the hidden section above', async ({ page, app }, info) => {
	await withDrag(page, app());
	await filter(page, ...open);
	await centre(header(page, 'Home'));
	const at = await box(row(page, 'Fix the sink'));
	await drag(page, info, 'Fix the sink', middle(await box(header(page, 'Empty'))) - at.y - at.height * 0.15, 800);
	await expect.poll(() => titles(page, 'Work')).toEqual(['Write the report', 'Review the slides', 'Later', 'Old ticket', 'Older ticket', 'Fix the sink']);
	await expect(list(page).getByText('No tasks')).toBeVisible();
	await expect(row(page, 'Later'), 'its separator shows with the row').toBeVisible();
	await expect(section(page, 'Later · 2 hidden')).toHaveCount(0);
});

test('a row dragged down past a folded header lands first there and unfolds it', async ({ page, app }, info) => {
	await withDrag(page, app());
	await list(page).getByRole('button', { name: 'Fold Work' }).click();
	await expect(row(page, 'Write the report')).toHaveCount(0);
	const at = await box(row(page, 'Renew passport'));
	await drag(page, info, 'Renew passport', middle(await box(header(page, 'Work'))) - at.y - at.height * 0.85);
	await expect.poll(() => titles(page, 'Work')).toEqual(['Renew passport', 'Write the report', 'Review the slides', 'Later', 'Old ticket', 'Older ticket']);
	await expect(row(page, 'Write the report')).toBeVisible();
});

test('a subtask lifts its parent, and the row moves with its subtasks', async ({ page, app }, info) => {
	await withDrag(page, app());
	await centre(header(page, 'Home'));
	const sub = await box(row(page, 'Buy the paint'));
	await hand(info)(page, { x: sub.x + sub.width / 2, from: middle(sub), to: middle(sub) - (await box(row(page, 'Fix the sink'))).height });
	await expect.poll(() => titles(page, 'Home')).toEqual(['Paint the door', 'Fix the sink', 'Mow the lawn', 'Clean the gutter']);
	const home = (await file(page)).categories.find((c) => c.name === 'Home')!.tasks.map((t) => t.title);
	expect(home.slice(0, 3)).toEqual(['Paint the door', 'Buy the paint', 'Sand the door']);
});

test('the list scrolls while a row is held at its bottom edge, and the row lands where it shows', async ({ page, app }, info) => {
	await withDrag(page, app());
	const at = await box(row(page, 'Plan the trip'));
	const bottom = page.viewportSize()!.height - 4;
	await hand(info)(page, { x: at.x + at.width / 2, from: middle(at), to: bottom, hold: 1500 });
	await expect.poll(async () => (await titles(page, 'Inbox')).includes('Plan the trip')).toBe(false);
	const scrolled = await page.evaluate(() => Math.max(scrollY, document.querySelector('section[aria-label="Tasks"]')!.scrollTop));
	expect(scrolled).toBeGreaterThan(200);
	// The row above it on the list is the one above it in the file.
	const above = await row(page, 'Plan the trip').evaluate((el) => {
		const prev = el.closest('[role="listitem"]')!.previousElementSibling!;
		return prev.querySelector('h2')?.textContent?.trim() ?? prev.querySelector('a')?.textContent?.trim() ?? '';
	});
	const project = await file(page);
	const into = project.categories.find((c) => c.tasks.some((t) => t.title === 'Plan the trip'))!;
	const tops = into.tasks.filter((t) => !t.parent_id).map((t) => t.title);
	const k = tops.indexOf('Plan the trip');
	expect(k === 0 ? above.endsWith(into.name) : above.startsWith(tops[k - 1]), `${above} above it`).toBe(true);
});

test('one undo takes a drop back, and the keys still move rows', async ({ page, app }, info) => {
	test.skip(phone(info), 'the keys are the wide screen’s');
	await withDrag(page, app());
	const before = await titles(page, 'Inbox');
	await drag(page, info, 'Renew passport', middle(await box(row(page, 'Plan the trip'))) - middle(await box(row(page, 'Renew passport'))));
	await expect.poll(() => titles(page, 'Inbox')).toEqual(['Renew passport', ...before.filter((t) => t !== 'Renew passport')]);
	await expect(page.getByRole('article', { name: 'Task' })).toBeHidden();
	await page.keyboard.press('u');
	await expect.poll(() => titles(page, 'Inbox')).toEqual(before);
	await page.keyboard.press('K');
	await expect.poll(() => titles(page, 'Inbox')).toEqual(['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Renew passport', 'Call the bank']);
});

test('a category drags among its siblings with its sub-categories', async ({ page, app }, info) => {
	await withDrag(page, app());
	await page.goto(app(`/p/${pid}/categories`));
	const page_ = page.getByRole('article', { name: 'Categories' });
	const shapes = await box(page_.getByRole('button', { name: 'Shapes', exact: true }));
	const empty = await box(page_.getByRole('button', { name: 'Empty', exact: true }));
	await expect(page_.getByRole('list', { name: 'Shapes sub-categories' }).getByRole('button', { name: 'Sub', exact: true })).toBeVisible();
	await hand(info)(page, { x: shapes.x + 20, from: middle(shapes), to: middle(empty) });
	await expect.poll(async () => (await file(page)).categories.map((c) => c.name)).toEqual(['Inbox', 'Work', 'Shapes', 'Sub', 'Empty', 'Home', 'Reading']);
	await expect(page).toHaveURL(app(`/p/${pid}/categories`));
	await shoot(page, 'drag-categories', info.project.name);
});

test('a sub-category drags among its siblings and stays under its parent', async ({ page, app }, info) => {
	await withDrag(page, app());
	await page.goto(app(`/p/${pid}/categories`));
	const page_ = page.getByRole('article', { name: 'Categories' });
	await page_.getByRole('button', { name: 'Edit Shapes' }).click();
	await page.getByRole('button', { name: 'New sub-category' }).click();
	await page.getByRole('textbox', { name: 'New sub-category' }).fill('Second');
	await page.keyboard.press('Enter');
	await expect(page.locator('dialog[open]')).toHaveCount(0);
	const subs = page_.getByRole('list', { name: 'Shapes sub-categories' });
	await expect(subs.getByRole('button', { name: 'Second', exact: true })).toBeVisible();
	const second = await box(subs.getByRole('button', { name: 'Second', exact: true }));
	const sub = await box(subs.getByRole('button', { name: 'Sub', exact: true }));
	await hand(info)(page, { x: second.x + 20, from: middle(second), to: middle(sub) });
	await expect
		.poll(async () => (await file(page)).categories.map((c) => c.name))
		.toEqual(['Inbox', 'Work', 'Empty', 'Home', 'Shapes', 'Second', 'Sub', 'Reading']);
	expect((await file(page)).categories.filter((c) => c.parent_id).map((c) => c.name)).toEqual(['Second', 'Sub']);
	await expect(page).toHaveURL(app(`/p/${pid}/categories`));
});

test.describe('a real touch', () => {
	test.use({ hasTouch: true });

	test('held on a row lifts it and the list doesn’t pan under it', async ({ page, app, browserName }, info) => {
		test.skip(browserName !== 'chromium' || !phone(info), 'touch input reaches only Chromium, through its protocol');
		await withDrag(page, app());
		const cdp = await page.context().newCDPSession(page);
		const at = await box(row(page, 'Renew passport'));
		const to = middle(await box(row(page, 'Plan the trip')));
		const x = at.x + at.width / 2;
		await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y: middle(at) }] });
		await page.waitForTimeout(700);
		for (let i = 1; i <= 20; i++) {
			await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x, y: middle(at) + ((to - middle(at)) * i) / 20 }] });
			await page.waitForTimeout(30);
		}
		await page.waitForTimeout(300);
		await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
		await expect.poll(() => titles(page, 'Inbox')).toEqual(['Renew passport', 'Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Call the bank']);
		expect(await page.evaluate(() => scrollY)).toBe(0);
		await expect(page).toHaveURL(app(`/p/${pid}`));
	});
});
