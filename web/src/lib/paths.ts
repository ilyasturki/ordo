/** The public build: the landing at /, the app at /app. The private one serves the app at /. */
export const site = import.meta.env.ORDO_WEB === 'site';

const root = site ? '/app' : '';

/** The URL of an app page, path written from the app's root: `app('/p/x')`. */
export function app(path = '/'): string {
	const url = root + path;
	return url.length > 1 && url.endsWith('/') ? url.slice(0, -1) : url;
}

/** The route an app URL lands on: src/routes/app holds the app in both builds. */
export function route(pathname: string): string | undefined {
	if (site) {
		return undefined;
	}
	return pathname === '/' ? '/app' : '/app' + pathname;
}
