import type { Page } from '@playwright/test';
import { Api } from '../src/lib/sync/api';
import { expect, shoot, test } from './fixtures';
import { ferry, row, taskLink } from './ordo';
import { passphrase, type Served } from './serve';

test.use({ signedIn: false });

// Signing in or out starts the page again, which drops whatever it was still loading.
test.beforeEach(({ noise }) => noise(/NS_BINDING_ABORTED|net::ERR_ABORTED|cancelled/));

type Project = { id: string; name: string; categories: { id: string; tasks: { id: string; title: string; status: string }[] }[] };

/** Another client of the same server, as the phone or the TUI. */
async function other(s: Served) {
	const api = new Api({ server: s.url, token: await s.token(), cookie: false });
	return {
		api,
		async projects(): Promise<Project[]> {
			return (await api.sync([], {}, {})).projects.map((p) => p.data as unknown as Project);
		},
		async send(project: string, op: Record<string, unknown>) {
			const answer = await api.sync([{ id: crypto.randomUUID(), project, action: '', ...op }], {}, {});
			expect(answer.results[0].status).toBe('applied');
		}
	};
}

async function shellStatus(them: Awaited<ReturnType<typeof other>>): Promise<string | undefined> {
	const [p] = await them.projects();
	return p.categories.flatMap((c) => c.tasks).find((t) => t.title.startsWith('Shell completions'))?.status;
}

async function signInWithPassphrase(page: Page, app: (p?: string) => string, build: string, s: Served) {
	await page.goto(app(build === 'site' ? '/signin' : '/'));
	if (build === 'site') await page.getByLabel('Server address').fill(s.url);
	await page.getByLabel('Passphrase').fill(passphrase);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();
}

test('signs in with the passphrase and works on the server’s projects', async ({ page, app, build, served }, info) => {
	served.add(ferry);
	await page.goto(app(build === 'site' ? '/signin' : '/'));
	if (build === 'site') await page.getByLabel('Server address').fill(served.url);
	await expect(page.getByLabel('Passphrase')).toBeVisible();
	await shoot(page, 'sign-in', info.project.name);
	await signInWithPassphrase(page, app, build, served);
	await row(page, 'Shell completions').getByRole('button', { name: 'Todo, change status' }).click();
	const them = await other(served);
	await expect.poll(() => shellStatus(them)).toBe('in_progress');
	await page.reload();
	await expect(row(page, 'Shell completions').getByRole('button', { name: 'In Progress, change status' })).toBeVisible();
});

test('a wrong passphrase says so', async ({ page, app, build, served, noise }) => {
	served.add(ferry);
	noise(/403/);
	await page.goto(app(build === 'site' ? '/signin' : '/'));
	if (build === 'site') await page.getByLabel('Server address').fill(served.url);
	await page.getByLabel('Passphrase').fill('not the passphrase');
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByText('Wrong passphrase')).toBeVisible();
});

test('pairs with a pairing link', async ({ page, app, build, served, noise }) => {
	served.add(ferry);
	const link = served.pairLink();
	const code = link.slice(link.lastIndexOf('/') + 1);
	if (build === 'private') {
		await page.goto(app(`/pair/${code}`));
		await expect(page.getByRole('heading', { name: 'Pair this browser?' })).toBeVisible();
		await expect(page).not.toHaveURL(new RegExp(code));
		await page.getByRole('button', { name: 'Pair' }).click();
	} else {
		await page.goto(app('/signin'));
		await page.getByLabel('Pairing link or code').fill(link);
		await expect(page.getByLabel('Server address')).toHaveValue(served.url);
		await page.getByRole('button', { name: 'Pair', exact: true }).click();
	}
	await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();
	if (build === 'private') {
		noise(/403/);
		await page.goto(app(`/pair/${code}`));
		await page.getByRole('button', { name: 'Pair' }).click();
		await expect(page.getByRole('alert')).toContainText('used or expired');
	}
});

test('an edit made offline reaches the server once back online', async ({ page, context, app, build, served, noise }) => {
	served.add(ferry);
	await signInWithPassphrase(page, app, build, served);
	noise(/ERR_INTERNET_DISCONNECTED|NS_ERROR_OFFLINE|WebKit encountered an internal error|CORS request did not succeed/);
	await context.setOffline(true);
	await row(page, 'Shell completions').getByRole('button', { name: 'Todo, change status' }).click();
	await expect(row(page, 'Shell completions').getByRole('button', { name: 'In Progress, change status' })).toBeVisible();
	const them = await other(served);
	await page.waitForTimeout(1000);
	expect(await shellStatus(them)).toBe('todo');
	await context.setOffline(false);
	await expect.poll(() => shellStatus(them), { timeout: 10_000 }).toBe('in_progress');
});

test('a change from another client shows up live', async ({ page, app, build, served }) => {
	served.add(ferry);
	await signInWithPassphrase(page, app, build, served);
	const them = await other(served);
	const [p] = await them.projects();
	const task = p.categories[0].tasks[0];
	await page.waitForTimeout(500);
	await them.send(p.id, { action: 'task.title', ids: [task.id], title: 'Renamed on the phone', base: task.title });
	await expect(taskLink(page, 'Renamed on the phone')).toBeVisible({ timeout: 5000 });
});

test('Devices pairs a new device and signs others out', async ({ page, app, build, served }, info) => {
	served.add(ferry);
	const them = await other(served);
	await signInWithPassphrase(page, app, build, served);
	await page.goto(app('/devices'));
	const devices = page.getByRole('list', { name: 'Devices' });
	await expect(devices.getByText('· this device')).toBeVisible();
	await expect(devices.getByText('Test client')).toBeVisible();
	await shoot(page, 'devices', info.project.name);

	await page.getByRole('button', { name: /Pair a new device/ }).click();
	await page.getByLabel('Passphrase').fill(passphrase);
	await page.getByRole('button', { name: 'Make a code' }).click();
	await expect(page.getByRole('img', { name: 'Pairing code' })).toBeVisible();
	await expect(page.getByText(/\/pair\//)).toBeVisible();
	await shoot(page, 'pair-code', info.project.name);
	await page.keyboard.press('Escape');

	await page.getByRole('button', { name: 'Sign out Test client' }).first().click();
	await page.getByRole('dialog').getByRole('button', { name: 'Sign out' }).click();
	await expect(devices.getByText('Test client')).toHaveCount(0);
	await expect(them.api.devices()).rejects.toThrow('Sign in again');
});

test('signing this browser out drops its copy', async ({ page, app, build, served }) => {
	served.add(ferry);
	await signInWithPassphrase(page, app, build, served);
	await page.goto(app('/devices'));
	await page.getByRole('button', { name: /^Sign out Drops/ }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Sign out' }).click();
	if (build === 'private') await expect(page.getByLabel('Pairing link or code')).toBeVisible();
	else await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	expect((await (await other(served)).api.devices()).devices.map((d) => d.name)).toEqual(['Test client']);
});

test('a browser the server signed out asks to sign in again, and keeps its edits', async ({ page, app, build, served, noise }) => {
	served.add(ferry);
	await signInWithPassphrase(page, app, build, served);
	const them = await other(served);
	const me = (await them.api.devices()).devices.find((d) => d.name !== 'Test client')!;
	await them.api.revoke(me.id);
	noise(/401/);
	await row(page, 'Shell completions').getByRole('button', { name: 'Todo, change status' }).click();
	await page.getByRole('button', { name: 'Not synced' }).click();
	await expect(page.getByRole('dialog', { name: 'Sync' })).toContainText('1 edit waits safely in this browser');
	await page.getByRole('button', { name: 'Sign in again' }).click();
	if (build === 'site') await expect(page.getByLabel('Server address')).toHaveValue(served.url);
	await page.getByLabel('Passphrase').fill(passphrase);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByRole('list', { name: 'ferry' })).toBeVisible();
	await expect.poll(() => shellStatus(them), { timeout: 10_000 }).toBe('in_progress');
});
