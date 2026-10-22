// Drives headless Chromium over the DevTools protocol: node shots.mjs <base-url> <cookie> <out-dir> [shot ...]
// ORDO_WEB_SHOTS names a module exporting another shot list, relative to the working directory.
import { spawn } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

const [base, cookie, out, ...only] = process.argv.slice(2);
// Each shot: a page, then optional script steps run in the page before the capture, then an optional
// touch gesture held down through it: a finger on `at` (a selector's center) or `point`, held `hold` ms, moved by `moves`,
// lifted when `release` is set.
// fold: the ordo_fold cookie.
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
// Runtime.evaluate reports a thrown script, such as a selector that matched nothing, as exceptionDetails rather than an error.
const evaluate = async (sessionId, expression, opts = {}) => {
  const { result, exceptionDetails } = await send("Runtime.evaluate", { expression, ...opts }, sessionId);
  if (exceptionDetails) throw new Error(`${expression}\n  ${exceptionDetails.exception?.description || exceptionDetails.text}`);
  return result;
};

async function gesture(sessionId, g) {
  let [x, y] = g.point || (await evaluate(sessionId,
    `(() => { const r = document.querySelector(${JSON.stringify(g.at)}).getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2]; })()`,
    { returnByValue: true })).value;
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
    await send("Page.enable", {}, sessionId);
    await send("Page.addScriptToEvaluateOnNewDocument", {
      source: "window.cspViolations = []; document.addEventListener('securitypolicyviolation', (e) => cspViolations.push(e.violatedDirective + ' ' + (e.sample || e.blockedURI)));",
    }, sessionId);
    await send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 2, mobile: true }, sessionId);
    await send("Emulation.setTouchEmulationEnabled", { enabled: true }, sessionId);
    await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: scheme }] }, sessionId);
    if (!shot.anon) await send("Network.setCookie", { name: "ordo_device", value: cookie, domain: host, path: "/" }, sessionId);
    if (shot.fold) await send("Network.setCookie", { name: "ordo_fold", value: shot.fold, domain: host, path: "/" }, sessionId);
    try {
      const loaded = until(sessionId, "Page.loadEventFired");
      await send("Page.navigate", { url: base + shot.path }, sessionId);
      await loaded;
      await sleep(300);
      for (const step of shot.steps || []) {
        await evaluate(sessionId, step, { awaitPromise: true });
        await sleep(350);
      }
      if (shot.gesture) await gesture(sessionId, shot.gesture);
      await sleep(shot.wait || 250);
      const { data } = await send("Page.captureScreenshot", { format: "png" }, sessionId);
      const file = join(out, `${shot.name}_${scheme}${width === 412 ? "" : "_" + width}.png`);
      writeFileSync(file, Buffer.from(data, "base64"));
      console.log(file);
      const result = await evaluate(sessionId, "window.cspViolations", { returnByValue: true });
      for (const v of result.value || []) violations.push(`${shot.name} (${scheme}): ${v}`);
      for (const step of shot.after || []) {
        await evaluate(sessionId, step);
        await sleep(600);
      }
    } catch (e) {
      failures.push(`${shot.name} (${scheme}): ${e.message}`);
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
if (failures.length) {
  console.error("Failed shots:\n" + failures.join("\n"));
  process.exitCode = 1;
}
