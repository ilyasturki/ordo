import type { Locator, Page, TestInfo } from '@playwright/test';
import type { Project } from '../src/lib/core/types';
import { expect, shoot, test } from './fixtures';
import { phone } from './ordo';

const fixture = new URL('../../testdata/native/drag.json', import.meta.url).pathname;
const pid = '0rd0-web-fixture';

async function withDrag(page: Page, url: string): Promise<void> {
	await page.goto(url);
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	const chooser = page.waitForEvent('filechooser');
	await page.getByRole('button', { name: 'Import a project' }).click();
	await (await chooser).setFiles(fixture);
	await expect(list(page)).toBeVisible();
}

const list = (page: Page) => page.getByRole('list', { name: 'Drag' });

const row = (page: Page, title: string) =>
	list(page)
		.locator('[data-key^="t:"]')
		.filter({ has: page.getByRole('link', { name: new RegExp('^' + title) }) });

const header = (page: Page, name: string) => list(page).getByRole('button', { name: `${name}, category menu` });

/** The project as this browser keeps it. */
function file(page: Page): Promise<Project> {
	return page.evaluate(
		(pid) =>
			new Promise<Project>((ok) => {
				const open = indexedDB.open('ordo');
				open.onsuccess = () => {
					const get = open.result.transaction('projects').objectStore('projects').get(pid);
					get.onsuccess = () => {
						open.result.close();
						ok(JSON.parse(get.result.raw));
					};
				};
			}),
		pid
	);
}

const titles = async (page: Page, category: string) =>
	(await file(page)).categories.find((c) => c.name === category)!.tasks.filter((t) => !t.parent_id).map((t) => t.title);

async function box(locator: Locator) {
	await expect(locator).toBeVisible();
	return (await locator.boundingBox())!;
}

async function filter(page: Page, ...chips: string[]) {
	await page.getByRole('button', { name: 'Filter', exact: true }).click();
	const sheet = page.getByRole('dialog', { name: 'Filter' });
	for (const chip of chips) await sheet.getByRole('button', { name: chip, exact: true }).click();
	await page.keyboard.press('Escape');
	await expect(sheet).toBeHidden();
	await expect(page.getByText('2 hidden by the filter').first()).toBeVisible();
}

const open = ['Todo', 'In Progress'];

type Drag = { x: number; from: number; to: number; hold?: number; held?: () => Promise<void> };

/** A mouse pressed on the row, moved by steps to the point and held there before the release. */
async function mouse(page: Page, { x, from, to, hold = 300, held }: Drag) {
	await page.mouse.move(x, from);
	await page.mouse.down();
	await page.mouse.move(x, from + 6, { steps: 2 });
	await page.mouse.move(x, to, { steps: 20 });
	await page.waitForTimeout(hold);
	await held?.();
	await page.mouse.up();
}

/** A finger held on the row until it lifts, then slid to the point, as the browser reports its pointer. */
async function finger(page: Page, d: Drag) {
	await page.evaluate(async ({ x, from, to, hold = 300 }) => {
		const fire = (type: string, y: number) =>
			(document.elementFromPoint(x, y) ?? document.body).dispatchEvent(
				new PointerEvent(type, {
					bubbles: true,
					cancelable: true,
					composed: true,
					pointerId: 7,
					pointerType: 'touch',
					isPrimary: true,
					clientX: x,
					clientY: y,
					button: type === 'pointermove' ? -1 : 0,
					buttons: type === 'pointerup' ? 0 : 1
				})
			);
		const wait = (ms: number) => new Promise((ok) => setTimeout(ok, ms));
		fire('pointerdown', from);
		await wait(600);
		for (let i = 1; i <= 20; i++) {
			fire('pointermove', from + ((to - from) * i) / 20);
			await wait(30);
		}
		await wait(hold);
		fire('pointerup', to);
	}, { x: d.x, from: d.from, to: d.to, hold: d.hold });
}

const hand = (info: TestInfo) => (phone(info) ? finger : mouse);

/** Drags the row by dy, from its middle; a mouse calls held before letting go. */
async function drag(page: Page, info: TestInfo, title: string, dy: number, hold?: number, held?: () => Promise<void>) {
	const at = await box(row(page, title));
	await hand(info)(page, { x: at.x + at.width / 2, from: at.y + at.height / 2, to: at.y + at.height / 2 + dy, hold, held });
}

const middle = (b: { y: number; height: number }) => b.y + b.height / 2;

/** Scrolls the line to the middle of the view, clear of the edges that scroll the list under a lifted row. */
const centre = (locator: Locator) => locator.evaluate((el) => el.scrollIntoView({ block: 'center' }));

for (const [name, chips] of [
	['a status filter', open],
	['a tag filter', ['Green']]
] as const) {
	test(`under ${name} a row dropped just above a separator stays in the section shown`, async ({ page, app }, info) => {
		await withDrag(page, app());
		await filter(page, ...chips);
		await expect(row(page, 'Done')).toHaveCount(0);
		await drag(page, info, 'Call the bank', middle(await box(row(page, 'Next'))) - middle(await box(row(page, 'Call the bank'))));
		await expect
			.poll(() => titles(page, 'Inbox'))
			.toEqual(['Plan the trip', 'Call the bank', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Renew passport']);
		await expect(row(page, 'Done'), 'no separator appears').toHaveCount(0);
		await expect(page).toHaveURL(app(`/p/${pid}`));
	});
}

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
	await expect(row(page, 'Later'), 'no separator appears').toHaveCount(0);
});

test('under a filter a row dragged up across a header stays above the hidden line', async ({ page, app }, info) => {
	await withDrag(page, app());
	await filter(page, ...open);
	const at = await box(row(page, 'Write the report'));
	await drag(page, info, 'Write the report', middle(await box(header(page, 'Work'))) - at.y - at.height * 0.15, undefined, () =>
		shoot(page, 'drag-lifted', info.project.name)
	);
	await expect
		.poll(() => titles(page, 'Inbox'))
		.toEqual(['Plan the trip', 'Done', 'Book the ferry', 'Pack the bags', 'Next', 'Call the bank', 'Renew passport', 'Write the report']);
	const hidden = page.getByText('2 hidden by the filter').first();
	expect((await box(row(page, 'Write the report'))).y + at.height).toBeLessThanOrEqual((await box(hidden)).y + 1);
	await shoot(page, 'drag-across-header', info.project.name);
});

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

test('under a filter a row dragged up through an empty category leaves its no tasks line and lands above', async ({ page, app }, info) => {
	await withDrag(page, app());
	await filter(page, ...open);
	await centre(header(page, 'Home'));
	const at = await box(row(page, 'Fix the sink'));
	await drag(page, info, 'Fix the sink', middle(await box(header(page, 'Empty'))) - at.y - at.height * 0.15, 800);
	await expect.poll(() => titles(page, 'Work')).toEqual(['Write the report', 'Review the slides', 'Fix the sink', 'Later', 'Old ticket', 'Older ticket']);
	await expect(list(page).getByText('No tasks')).toBeVisible();
	await expect(row(page, 'Later')).toHaveCount(0);
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
