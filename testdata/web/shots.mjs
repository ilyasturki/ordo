// Drives headless Chromium over the DevTools protocol: node shots.mjs <base-url> <cookie> <out-dir> [shot ...]
// ORDO_WEB_SHOTS names a module exporting another shot list, relative to the working directory.
import { spawn } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

const [base, cookie, out, ...only] = process.argv.slice(2);
// Each shot: a page, opened once `ready` (a selector, [data-row] by default) matches, then steps run before the capture.
// A string step is a script run in the page; an object step is a finger on `at` (the center of the first match holding
// the text `has`, if set) or `point`, held `hold` ms, moved by `moves`, lifted when `release` is set, or `text` typed
// into the focused field.
// prefs: the app's ordo.prefs in localStorage. anon: no device cookie. after: steps run once the capture is taken.
const shots = (await import(process.env.ORDO_WEB_SHOTS ? pathToFileURL(resolve(process.env.ORDO_WEB_SHOTS)).href : "./fixture-shots.mjs")).default;

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
  } else for (const l of [...listeners]) l(msg);
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
// Runtime.evaluate reports a thrown script, such as a selector that matched nothing, as exceptionDetails rather than an error.
const evaluate = async (sessionId, expression, opts = {}, label = expression) => {
  const { result, exceptionDetails } = await send("Runtime.evaluate", { expression, ...opts }, sessionId);
  if (exceptionDetails) throw new Error(`${label}\n  ${exceptionDetails.exception?.description || exceptionDetails.text}`);
  return result;
};

async function waitFor(sessionId, selector, ms = 15000) {
  const check = `!!document.querySelector(${JSON.stringify(selector)})`;
  for (const end = Date.now() + ms; Date.now() < end; await sleep(100)) {
    if ((await evaluate(sessionId, check, { returnByValue: true })).value) return;
  }
  throw new Error(`${selector} did not show within ${ms} ms`);
}

// The finger lands where a user would see the element: scrolled into view, and on top at that point.
const centerOf = (sel, has = "") => `(() => {
  const el = [...document.querySelectorAll(${JSON.stringify(sel)})].find((e) => e.textContent.includes(${JSON.stringify(has)}));
  if (!el) throw new Error('nothing matches ' + ${JSON.stringify(sel + (has && ` holding “${has}”`))});
  let r = el.getBoundingClientRect();
  if (r.top < 0 || r.bottom > innerHeight) { el.scrollIntoView({ block: 'center' }); r = el.getBoundingClientRect(); }
  const x = r.left + r.width / 2, y = r.top + r.height / 2;
  if (!el.contains(document.elementFromPoint(x, y))) throw new Error(${JSON.stringify(sel)} + ' is covered at its center');
  return [x, y];
})()`;

async function gesture(sessionId, g) {
  let [x, y] = g.point || (await evaluate(sessionId, centerOf(g.at, g.has), { returnByValue: true }, `touch ${g.at}`)).value;
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
  if (g.release) await send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] }, sessionId);
}

async function run(sessionId, step) {
  if (typeof step === "string") await evaluate(sessionId, step, { awaitPromise: true });
  else if (step.text !== undefined) await send("Input.insertText", { text: step.text }, sessionId);
  else await gesture(sessionId, step);
}

const host = new URL(base).hostname;
const violations = [];
const failures = [];
const [width, height] = (process.env.ORDO_WEB_SIZE || "412x915").split("x").map(Number);
for (const scheme of ["light", "dark"]) {
  for (const shot of shots) {
    if (only.length && !only.includes(shot.name)) continue;
    const { browserContextId } = await send("Target.createBrowserContext");
    const { targetId } = await send("Target.createTarget", { url: "about:blank", browserContextId });
    const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
    const errors = [];
    const caught = (m) => {
      if (m.sessionId === sessionId && m.method === "Runtime.exceptionThrown") {
        const d = m.params.exceptionDetails;
        errors.push(d.exception?.description || d.text);
      }
    };
    listeners.push(caught);
    await send("Page.enable", {}, sessionId);
    await send("Runtime.enable", {}, sessionId);
    await send("Page.addScriptToEvaluateOnNewDocument", {
      source: "window.cspViolations = []; document.addEventListener('securitypolicyviolation', (e) => cspViolations.push(e.violatedDirective + ' ' + (e.sample || e.blockedURI)));" +
        (shot.prefs ? `localStorage.setItem('ordo.prefs', ${JSON.stringify(JSON.stringify(shot.prefs))});` : ""),
    }, sessionId);
    await send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 2, mobile: true }, sessionId);
    await send("Emulation.setTouchEmulationEnabled", { enabled: true }, sessionId);
    await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: scheme }] }, sessionId);
    if (!shot.anon) await send("Network.setCookie", { name: "ordo_device", value: cookie, domain: host, path: "/", httpOnly: true }, sessionId);
    try {
      const loaded = until(sessionId, "Page.loadEventFired");
      await send("Page.navigate", { url: base + shot.path }, sessionId);
      await loaded;
      await waitFor(sessionId, shot.ready || "[data-row]");
      await sleep(300);
      for (const step of shot.steps || []) {
        await run(sessionId, step);
        await sleep(350);
      }
      await sleep(shot.wait || 250);
      const { data } = await send("Page.captureScreenshot", { format: "png" }, sessionId);
      const file = join(out, `${shot.name}_${scheme}${width === 412 ? "" : "_" + width}.png`);
      writeFileSync(file, Buffer.from(data, "base64"));
      console.log(file);
      const result = await evaluate(sessionId, "window.cspViolations", { returnByValue: true });
      for (const v of result.value || []) violations.push(`${shot.name} (${scheme}): ${v}`);
      for (const step of shot.after || []) {
        await run(sessionId, step);
        await sleep(600);
      }
      for (const e of errors) failures.push(`${shot.name} (${scheme}): uncaught ${e}`);
    } catch (e) {
      failures.push(`${shot.name} (${scheme}): ${e.message}`);
    }
    listeners.splice(listeners.indexOf(caught), 1);
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
if (failures.length) {
  console.error("Failed shots:\n" + failures.join("\n"));
  process.exitCode = 1;
}
