import { createReadStream, existsSync, statSync } from 'node:fs';
import { createServer, request, type Server } from 'node:http';
import { extname, join, normalize } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../build/private', import.meta.url));

/** The policy owed/web-sync.md hands the deploy, with connect-src for the server the site build may reach. */
export function csp(connect = ''): string {
	return [
		"default-src 'self'",
		"script-src 'self' 'wasm-unsafe-eval' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' blob: data:",
		"font-src 'self'",
		`connect-src 'self'${connect ? ' ' + connect : ''}`,
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'none'",
		"frame-ancestors 'none'"
	].join('; ');
}

const types: Record<string, string> = {
	'.html': 'text/html; charset=utf-8',
	'.js': 'text/javascript',
	'.css': 'text/css',
	'.wasm': 'application/wasm',
	'.svg': 'image/svg+xml',
	'.woff2': 'font/woff2',
	'.json': 'application/json',
	'.png': 'image/png'
};

/** The private build as Caddy serves it on the home server: the static app, /api proxied to ordo serve on the same origin. */
export async function front(api: string): Promise<{ url: string; close(): Promise<void> }> {
	const target = new URL(api);
	const server: Server = createServer((req, res) => {
		const path = decodeURIComponent((req.url ?? '/').split('?')[0]);
		if (path.startsWith('/api/')) {
			const out = request(
				{ host: target.hostname, port: target.port, method: req.method, path: req.url, headers: { ...req.headers, host: req.headers.host } },
				(answer) => {
					res.writeHead(answer.statusCode ?? 502, answer.headers);
					answer.pipe(res);
				}
			);
			out.on('error', () => {
				if (!res.headersSent) res.writeHead(502);
				res.end();
			});
			req.pipe(out);
			res.on('close', () => out.destroy());
			return;
		}
		let file = join(root, normalize(path));
		if (!file.startsWith(root) || !existsSync(file) || statSync(file).isDirectory()) file = join(root, '200.html');
		res.writeHead(200, { 'Content-Type': types[extname(file)] ?? 'application/octet-stream', 'Content-Security-Policy': csp() });
		createReadStream(file).pipe(res);
	});
	await new Promise<void>((r) => server.listen(0, '127.0.0.1', r));
	return {
		url: `http://127.0.0.1:${(server.address() as { port: number }).port}`,
		close: () =>
			new Promise((r) => {
				server.closeAllConnections();
				server.close(() => r());
			})
	};
}
