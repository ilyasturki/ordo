import type { Page, TestInfo } from '@playwright/test';
import { expect } from './fixtures';

export const ferry = new URL('../../assets/demo/ferry.json', import.meta.url).pathname;

export const phone = (info: TestInfo) => info.project.name.endsWith('-phone');

/** Opens the app empty and imports the demo project, which it then shows. */
export async function withFerry(page: Page, url: string): Promise<void> {
	await page.goto(url);
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	const chooser = page.waitForEvent('filechooser');
	await page.getByRole('button', { name: 'Import a project' }).click();
	await (await chooser).setFiles(ferry);
	await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();
}

/** A value the app keeps in this browser's IndexedDB, under meta. */
export function stored(page: Page, key: string): Promise<unknown> {
	return page.evaluate(
		(key) =>
			new Promise((ok) => {
				const open = indexedDB.open('ordo');
				open.onsuccess = () => {
					const get = open.result.transaction('meta').objectStore('meta').get(key);
					get.onsuccess = () => {
						open.result.close();
						ok(get.result);
					};
				};
			}),
		key
	);
}

/** A task's row on the list, by the start of its title. */
export function row(page: Page, title: string) {
	return page.getByRole('list', { name: 'ferry' }).locator('[data-key^="t:"]').filter({ has: page.getByRole('link', { name: new RegExp('^' + title) }) });
}

export function taskLink(page: Page, title: string) {
	return page.getByRole('list', { name: 'ferry' }).getByRole('link', { name: new RegExp('^' + title) });
}

/** The task's page on a phone, its pane beside the list on a wide screen. */
export function pane(page: Page) {
	return page.getByRole('article', { name: 'Task' });
}

/** No part of the page reaches past the viewport's width. */
export async function fitsWidth(page: Page): Promise<void> {
	const over = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
	expect(over).toBeLessThanOrEqual(0);
}
