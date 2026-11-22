import type { Locator, Page, TestInfo } from '@playwright/test';
import type { Project } from '../src/lib/core/types';
import { expect } from './fixtures';
import { phone } from './ordo';

const fixture = new URL('../../testdata/native/drag.json', import.meta.url).pathname;
export const pid = '0rd0-web-fixture';

export async function withDrag(page: Page, url: string): Promise<void> {
	await page.goto(url);
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	const chooser = page.waitForEvent('filechooser');
	await page.getByRole('button', { name: 'Import a project' }).click();
	await (await chooser).setFiles(fixture);
	await expect(list(page)).toBeVisible();
}

export const list = (page: Page) => page.getByRole('list', { name: 'Drag' });

export const row = (page: Page, title: string) =>
	list(page)
		.locator('[data-key^="t:"]')
		.filter({ has: page.getByRole('link', { name: new RegExp('^' + title) }) });

export const header = (page: Page, name: string) => list(page).getByRole('button', { name: `${name}, category menu` });

/** A hidden section's line, by what it reads. */
export const section = (page: Page, text: string) => list(page).getByRole('button', { name: text, exact: true });

/** The project as this browser keeps it. */
export function file(page: Page): Promise<Project> {
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

export const titles = async (page: Page, category: string) =>
	(await file(page)).categories.find((c) => c.name === category)!.tasks.filter((t) => !t.parent_id).map((t) => t.title);

export async function box(locator: Locator) {
	await expect(locator).toBeVisible();
	return (await locator.boundingBox())!;
}

export async function filter(page: Page, ...chips: string[]) {
	await page.getByRole('button', { name: 'Filter', exact: true }).click();
	const sheet = page.getByRole('dialog', { name: 'Filter' });
	for (const chip of chips) await sheet.getByRole('button', { name: chip, exact: true }).click();
	await page.keyboard.press('Escape');
	await expect(sheet).toBeHidden();
	await expect(section(page, 'Done · 2 hidden')).toBeVisible();
}

export const open = ['Todo', 'In Progress'];

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

export const hand = (info: TestInfo) => (phone(info) ? finger : mouse);

/** Drags the row of that title, or the line, by dy from its middle; a mouse calls held before letting go. */
export async function drag(page: Page, info: TestInfo, what: string | Locator, dy: number, hold?: number, held?: () => Promise<void>) {
	const at = await box(typeof what === 'string' ? row(page, what) : what);
	await hand(info)(page, { x: at.x + at.width / 2, from: at.y + at.height / 2, to: at.y + at.height / 2 + dy, hold, held });
}

export const middle = (b: { y: number; height: number }) => b.y + b.height / 2;

/** Scrolls the line to the middle of the view, clear of the edges that scroll the list under a lifted row. */
export const centre = (locator: Locator) => locator.evaluate((el) => el.scrollIntoView({ block: 'center' }));
