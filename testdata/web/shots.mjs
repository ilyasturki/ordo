// Drives headless Chromium over the DevTools protocol: node shots.mjs <base-url> <cookie> <out-dir> [shot ...]
import { spawn } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const [base, cookie, out, ...only] = process.argv.slice(2);
const P = "/p/0rd0-web-fixture";
const T = (n) => `${P}/t/aaaaaaaa-0000-4000-8000-${String(n).padStart(12, "0")}`;
const C = (n) => `cccccccc-0000-4000-8000-${String(n).padStart(12, "0")}`;
const folded = C(24);
// A pointer click (detail 1), unlike element.click(), which reads as a keyboard press.
const tap = (sel) => `document.querySelector('${sel}').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, detail: 1 }))`;

// Each shot: a page, then optional script steps run in the page before the capture, then an optional
// touch gesture held down through it: a finger on `at` (a selector's center) or `point`, held `hold` ms, moved by `moves`.
// fold: the ordo_fold cookie.
const shots = [
  { name: "projects", path: "/projects" },
  { name: "project", path: P },
  { name: "project_folded", path: P, fold: folded },
  { name: "project_scrolled", path: P, steps: ["window.scrollTo(0, 700)"] },
  { name: "completed_open", path: P, fold: `done-${C(12)}.done-${C(19)}`, steps: [`document.getElementById('c-${C(12)}').scrollIntoView()`, "window.scrollBy(0, -72)"] },
  { name: "task", path: T(9) },
  { name: "task_parent", path: T(1) },
  { name: "task_child", path: T(3) },
  // Another device renames the task while the title field is focused, then the typed title is saved.
  { name: "task_conflict", path: T(9), wait: 900, steps: [
    "window.rename = (base, title) => fetch(document.querySelector('.title-form').action, { method: 'POST', credentials: 'same-origin', headers: { 'HX-Request': 'true' }, body: new URLSearchParams({ id: document.querySelector('#app').dataset.task, base, title }) }).then((r) => r.status)",
    "window.original = document.querySelector('.title-form [data-base]').value; document.querySelector('#title').focus()",
    "rename(original, 'Renamed on the laptop')",
    "(() => { const f = document.querySelector('#title'); f.value = 'Typed on the phone'; f.dispatchEvent(new Event('change', { bubbles: true })); f.blur(); })()",
  ], after: ["rename('Renamed on the laptop', original)"] },
  { name: "task_done", path: T(2) },
  { name: "task_status", path: T(1), steps: ["document.querySelector('[data-sheet-open=status-sheet]').click()"] },
  { name: "task_priority", path: T(1), steps: ["document.querySelector('[data-sheet-open=priority-sheet]').click()"] },
  { name: "task_tag", path: T(1), steps: ["document.querySelector('[data-sheet-open=tag-sheet]').click()"] },
  { name: "task_estimate", path: T(1), steps: ["document.querySelector('[data-sheet-open=estimate-sheet]').click()"] },
  { name: "task_move", path: T(3), steps: ["document.querySelector('[data-sheet-open=move-sheet]').click()"] },
  { name: "task_menu", path: T(3), steps: ["document.querySelector('[data-sheet-open=task-menu]').click()"] },
  { name: "task_typing", path: T(1), steps: ["document.querySelector('#new-subtask').focus()"] },
  { name: "task_swap", path: T(9), steps: ["htmx.ajax('GET', location.pathname, { target: '#app', select: '#app', swap: 'outerHTML' })"] },
  { name: "add_row", path: P, steps: ["document.querySelector('[data-add-fab]').click()", "document.querySelector('.composer-input').value = 'Offline mode'"] },
  { name: "cat_menu", path: P, steps: [tap(`#c-${C(12)} [data-cat-menu]`)] },
  { name: "separator_label", path: P, steps: [
    tap(`#c-${C(8)} [data-cat-menu]`),
    "document.querySelector('[data-menu=separator]').click()",
    "document.querySelector('.separator-input').value = 'Next release'",
  ], wait: 600, after: [
    "{ const i = document.querySelector('.separator-input'); i.value = ''; i.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })); }",
    "document.querySelector('.snackbar-action').click()",
  ] },
  { name: "select", path: P, steps: [
    "document.querySelector('[data-select-start]').click()",
    "document.querySelectorAll('.content .row .item-body')[0].click()",
    "document.querySelectorAll('.content .row .item-body')[4].click()",
  ] },
  { name: "bulk_sheet", path: P, steps: [
    "document.querySelector('[data-select-start]').click()",
    "document.querySelectorAll('.content .row .item-body')[0].click()",
    "document.querySelector('[data-sheet-open=bulk-sheet]').click()",
  ] },
  { name: "drawer", path: P, steps: ["document.querySelector('[data-sheet-open=drawer]').click()"] },
  { name: "drawer_swipe", path: P, gesture: { point: [12, 400], moves: [[30, 0], [80, 2], [80, 2]] } },
  { name: "sheet_swipe", path: P, steps: [
    "document.querySelector('[data-select-start]').click()",
    "document.querySelectorAll('.content .row .item-body')[0].click()",
    "document.querySelector('[data-sheet-open=bulk-sheet]').click()",
  ], gesture: { at: "#bulk-sheet .drag-handle", moves: [[0, 40], [0, 80], [0, 80]] } },
  { name: "drag_lift", path: P, gesture: { at: "#t-aaaaaaaa-0000-4000-8000-000000000005 > .list-item", hold: 600 } },
  { name: "drag_nest", path: P, gesture: { at: "#t-aaaaaaaa-0000-4000-8000-000000000005 > .list-item", hold: 600, moves: [[20, 0], [30, 4], [20, 0]] } },
  { name: "drag_separator", path: P, gesture: { at: "#t-bbbbbbbb-0000-4000-8000-000000000006", hold: 600, moves: [[0, -20], [0, -40], [0, -10]] } },
  { name: "drag_empty", path: P, steps: [
    "document.querySelector('#c-cccccccc-0000-4000-8000-000000000014 .category-body > .rows').replaceChildren()",
    "document.querySelector('#t-aaaaaaaa-0000-4000-8000-000000000009').scrollIntoView({ block: 'center' })",
  ], gesture: { at: "#t-aaaaaaaa-0000-4000-8000-000000000009 > .list-item", hold: 600, moves: [[0, 16], [0, 16]] } },
  { name: "snackbar", path: P, steps: ["document.querySelector('.status-button').click()"], wait: 900,
    after: ["document.querySelector('.snackbar-action').click()"] },
  { name: "devices", path: "/devices" },
  { name: "devices_pair", path: "/devices", steps: ["document.querySelector('form[action=\"/devices/pair\"] button').click()"], wait: 900 },
  { name: "login", path: "/login", anon: true },
  { name: "scan_nocamera", path: "/login", anon: true, steps: ["document.querySelector('[data-scan]').click()"], wait: 900 },
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

async function gesture(sessionId, g) {
  let [x, y] = g.point || (await send("Runtime.evaluate", {
    expression: `(() => { const r = document.querySelector(${JSON.stringify(g.at)}).getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2]; })()`,
    returnByValue: true,
  }, sessionId)).result.value;
  const touch = (type) => send("Input.dispatchTouchEvent", { type, touchPoints: [{ x, y }] }, sessionId);
  await touch("touchStart");
  await sleep(g.hold || 0);
  for (const [dx, dy] of g.moves || []) {
    for (let i = 0; i < 4; i++) {
      x += dx / 4;
      y += dy / 4;
      await touch("touchMove");
      await sleep(16);
    }
  }
}

const host = new URL(base).hostname;
const violations = [];
const [width, height] = (process.env.ORDO_WEB_SIZE || "412x915").split("x").map(Number);
for (const scheme of ["light", "dark"]) {
  for (const shot of shots) {
    if (only.length && !only.includes(shot.name)) continue;
    const { browserContextId } = await send("Target.createBrowserContext");
    const { targetId } = await send("Target.createTarget", { url: "about:blank", browserContextId });
    const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
    await send("Page.enable", {}, sessionId);
    await send("Page.addScriptToEvaluateOnNewDocument", {
      source: "window.cspViolations = []; document.addEventListener('securitypolicyviolation', (e) => cspViolations.push(e.violatedDirective + ' ' + (e.sample || e.blockedURI)));",
    }, sessionId);
    await send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 2, mobile: true }, sessionId);
    await send("Emulation.setTouchEmulationEnabled", { enabled: true }, sessionId);
    await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: scheme }] }, sessionId);
    if (!shot.anon) await send("Network.setCookie", { name: "ordo_device", value: cookie, domain: host, path: "/" }, sessionId);
    if (shot.fold) await send("Network.setCookie", { name: "ordo_fold", value: shot.fold, domain: host, path: "/" }, sessionId);
    const loaded = until(sessionId, "Page.loadEventFired");
    await send("Page.navigate", { url: base + shot.path }, sessionId);
    await loaded;
    await sleep(300);
    for (const step of shot.steps || []) {
      await send("Runtime.evaluate", { expression: step, awaitPromise: true }, sessionId);
      await sleep(350);
    }
    if (shot.gesture) await gesture(sessionId, shot.gesture);
    await sleep(shot.wait || 250);
    const { data } = await send("Page.captureScreenshot", { format: "png" }, sessionId);
    const file = join(out, `${shot.name}_${scheme}${width === 412 ? "" : "_" + width}.png`);
    writeFileSync(file, Buffer.from(data, "base64"));
    console.log(file);
    const { result } = await send("Runtime.evaluate", { expression: "window.cspViolations", returnByValue: true }, sessionId);
    for (const v of result.value || []) violations.push(`${shot.name} (${scheme}): ${v}`);
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
rmSync(profile, { recursive: true, force: true, maxRetries: 5 });
if (violations.length) {
  console.error("Content-Security-Policy violations:\n" + violations.join("\n"));
  process.exitCode = 1;
}
