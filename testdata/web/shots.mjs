// Drives headless Chromium over the DevTools protocol: node shots.mjs <base-url> <cookie> <out-dir> [shot ...]
import { spawn } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const [base, cookie, out, ...only] = process.argv.slice(2);
const P = "/p/0rd0-web-fixture";
const T = (n) => `${P}/t/aaaaaaaa-0000-4000-8000-${String(n).padStart(12, "0")}`;
const folded = "cccccccc-0000-4000-8000-000000000024";

// Each shot: a page, then optional script steps run in the page before the capture.
const shots = [
  { name: "projects", path: "/projects" },
  { name: "project", path: `${P}?f=all` },
  { name: "project_open", path: `${P}?f=open`, fold: true },
  { name: "project_scrolled", path: `${P}?f=all`, steps: ["window.scrollTo(0, 700)"] },
  { name: "task", path: T(9) },
  { name: "task_parent", path: T(1) },
  { name: "task_child", path: T(3) },
  { name: "task_done", path: T(2) },
  { name: "task_status", path: T(1), steps: ["document.querySelector('[data-sheet-open=status-sheet]').click()"] },
  { name: "task_priority", path: T(1), steps: ["document.querySelector('[data-sheet-open=priority-sheet]').click()"] },
  { name: "task_tag", path: T(1), steps: ["document.querySelector('[data-sheet-open=tag-sheet]').click()"] },
  { name: "task_estimate", path: T(1), steps: ["document.querySelector('[data-sheet-open=estimate-sheet]').click()"] },
  { name: "task_move", path: T(3), steps: ["document.querySelector('[data-sheet-open=move-sheet]').click()"] },
  { name: "task_menu", path: T(3), steps: ["document.querySelector('[data-sheet-open=task-menu]').click()"] },
  { name: "task_typing", path: T(1), steps: ["document.querySelector('#new-subtask').focus()"] },
  { name: "add_sheet", path: `${P}?f=open`, steps: ["document.querySelector('[data-fab]').click()"] },
  { name: "select", path: `${P}?f=open`, steps: [
    "document.querySelector('[data-select-start]').click()",
    "document.querySelectorAll('.content .row .item-body')[0].click()",
    "document.querySelectorAll('.content .row .item-body')[4].click()",
  ] },
  { name: "bulk_sheet", path: `${P}?f=open`, steps: [
    "document.querySelector('[data-select-start]').click()",
    "document.querySelectorAll('.content .row .item-body')[0].click()",
    "document.querySelector('[data-sheet-open=bulk-sheet]').click()",
  ] },
  { name: "drawer", path: `${P}?f=open`, steps: ["document.querySelector('[data-sheet-open=drawer]').click()"] },
  { name: "snackbar", path: `${P}?f=open`, steps: ["document.querySelector('.status-button').click()"], wait: 900,
    after: ["document.querySelector('.snackbar-action').click()"] },
  { name: "devices", path: "/devices" },
  { name: "login", path: "/login", anon: true },
  { name: "pair", path: "/pair/sample-token", anon: true },
];

const profile = mkdtempSync(join(tmpdir(), "ordo-shots-"));
const chrome = spawn(process.env.CHROMIUM || "chromium", [
  "--headless=new", "--remote-debugging-port=0", `--user-data-dir=${profile}`,
  "--no-first-run", "--no-default-browser-check", "--disable-gpu", "--hide-scrollbars", "about:blank",
], { stdio: "ignore" });

const portFile = join(profile, "DevToolsActivePort");
for (let i = 0; i < 100 && !existsSync(portFile); i++) await new Promise((r) => setTimeout(r, 100));
const [port, path] = readFileSync(portFile, "utf8").trim().split("\n");
const ws = new WebSocket(`ws://127.0.0.1:${port}${path}`);
await new Promise((r) => ws.addEventListener("open", r, { once: true }));

let seq = 0;
const pending = new Map();
const listeners = [];
ws.addEventListener("message", (e) => {
  const msg = JSON.parse(e.data);
  if (msg.id && pending.has(msg.id)) {
    const { resolve, reject } = pending.get(msg.id);
    pending.delete(msg.id);
    msg.error ? reject(new Error(msg.error.message)) : resolve(msg.result);
  } else for (const l of listeners) l(msg);
});
const send = (method, params = {}, sessionId) => new Promise((resolve, reject) => {
  const id = ++seq;
  pending.set(id, { resolve, reject });
  ws.send(JSON.stringify({ id, method, params, sessionId }));
});
const until = (sessionId, method) => new Promise((resolve) => {
  const l = (m) => { if (m.sessionId === sessionId && m.method === method) { listeners.splice(listeners.indexOf(l), 1); resolve(m); } };
  listeners.push(l);
});
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const host = new URL(base).hostname;
const [width, height] = (process.env.ORDO_WEB_SIZE || "412x915").split("x").map(Number);
for (const scheme of ["light", "dark"]) {
  for (const shot of shots) {
    if (only.length && !only.includes(shot.name)) continue;
    const { browserContextId } = await send("Target.createBrowserContext");
    const { targetId } = await send("Target.createTarget", { url: "about:blank", browserContextId });
    const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
    await send("Page.enable", {}, sessionId);
    await send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 2, mobile: true }, sessionId);
    await send("Emulation.setTouchEmulationEnabled", { enabled: true }, sessionId);
    await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: scheme }] }, sessionId);
    if (!shot.anon) await send("Network.setCookie", { name: "ordo_device", value: cookie, domain: host, path: "/" }, sessionId);
    if (shot.fold) await send("Network.setCookie", { name: "ordo_fold", value: folded, domain: host, path: "/" }, sessionId);
    const loaded = until(sessionId, "Page.loadEventFired");
    await send("Page.navigate", { url: base + shot.path }, sessionId);
    await loaded;
    await sleep(300);
    for (const step of shot.steps || []) {
      await send("Runtime.evaluate", { expression: step, awaitPromise: true }, sessionId);
      await sleep(350);
    }
    await sleep(shot.wait || 250);
    const { data } = await send("Page.captureScreenshot", { format: "png" }, sessionId);
    const file = join(out, `${shot.name}_${scheme}${width === 412 ? "" : "_" + width}.png`);
    writeFileSync(file, Buffer.from(data, "base64"));
    console.log(file);
    for (const step of shot.after || []) {
      await send("Runtime.evaluate", { expression: step }, sessionId);
      await sleep(600);
    }
    await send("Target.disposeBrowserContext", { browserContextId });
  }
}
ws.close();
chrome.kill();
await sleep(200);
rmSync(profile, { recursive: true, force: true });
