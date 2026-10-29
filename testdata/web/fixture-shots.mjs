// The dev fixture's shots (testdata/web/seed.json), the driver's default list.
const P = "0rd0-web-fixture";
const A = (n) => `aaaaaaaa-0000-4000-8000-${String(n).padStart(12, "0")}`;
const C = (n) => `cccccccc-0000-4000-8000-${String(n).padStart(12, "0")}`;
const T = (n) => `/p/${P}/t/${A(n)}`;
const row = (n) => `[data-row="${A(n)}"]`;
const cat = (n) => `[data-cat="${C(n)}"]`;
const tap = (at, has) => ({ at, has, release: true });
const press = (at) => ({ at, hold: 600, release: true });
const assert = (cond, msg = cond) => `(() => { if (!(${cond})) throw new Error(${JSON.stringify(msg)}); })()`;
const poll = (cond, msg = cond, ms = 5000) => `new Promise((ok, fail) => { const end = Date.now() + ${ms}; const t = setInterval(() => {
  if (${cond}) { clearInterval(t); ok(); } else if (Date.now() > end) { clearInterval(t); fail(new Error(${JSON.stringify(msg)})); } }, 50); })`;
const sheet = "[data-vaul-drawer][data-state=open]";
const shown = "[...document.querySelectorAll('[data-row]')].map((r) => r.dataset.row.slice(-2))";
const labels = (sel) => `[...document.querySelectorAll(${JSON.stringify(sel)})].map((e) => e.textContent.trim()).join()`;
const onScreen = (el) => `(({ top, left, right, bottom }) => top >= 0 && left >= 0 && right <= innerWidth && bottom <= innerHeight)(${el}.getBoundingClientRect())`;
const title = "textarea[aria-label=Title]";

export default [
  { name: "project", path: "/", steps: [
    assert(`document.querySelector('[data-tab="${P}"]').matches('[aria-current=page]') && document.querySelector('[data-cat]').matches('${cat(8)}') && !document.querySelector('${row(10)}')`,
      "the fixture is not the open tab with Feature first and Completed folded"),
  ] },
  { name: "project_folded", path: "/", prefs: { folded: { [C(8)]: true } }, steps: [
    assert(`!document.querySelector('${cat(8)} [data-open]') && document.querySelector('${cat(8)} [aria-expanded=false]') && document.querySelector('${cat(12)} [data-open]')`,
      "Feature is not the one folded category"),
  ] },
  { name: "project_scrolled", path: "/", steps: [
    "window.scrollTo(0, 700)",
    assert("scrollY > 600 && document.querySelector('header').getBoundingClientRect().top === 0", "the list did not scroll under a pinned header"),
  ] },
  { name: "completed_open", path: "/", prefs: { doneOpen: { [C(12)]: true, [C(19)]: true } }, steps: [
    `document.querySelector('${cat(12)}').scrollIntoView(); window.scrollBy(0, -120)`,
    assert(`['${row(10)}', '${row(11)}', '${row(15)}', '${row(17)}'].every((s) => document.querySelector(s)) && [...document.querySelectorAll('${cat(12)} button[aria-expanded=true]')].some((b) => b.textContent.includes('Completed (2)'))`,
      "Fix and Documentation do not show their completed tasks"),
  ] },
  { name: "task", path: T(9), ready: "main figure img", steps: [
    "document.querySelector('main figure img').decode()",
    assert(`document.querySelector('main figure img').naturalWidth > 0 && document.querySelector('${title}').value === 'Cursor jumps after a sync reload'`,
      "the title or the attached image did not load"),
  ] },
  { name: "task_parent", path: T(1), ready: title, steps: [
    assert(`${labels("main a[href*='/t/']")} === 'Shared action layer,Pairing link login,Drag to reorder'`, "the subtasks are not listed in order"),
  ] },
  { name: "task_child", path: T(3), ready: title, steps: [
    assert(`document.querySelector('main a[href="${T(1)}"]')?.textContent.includes('Subtask of Web UI for the phone') && !document.querySelector('main form')`,
      "a subtask does not link its parent, or offers subtasks of its own"),
  ] },
  { name: "task_done", path: T(2), ready: title, steps: [
    assert(`document.querySelector('${title}').matches('.line-through')`, "a done task's title is not struck through"),
  ] },
  { name: "task_move", path: T(3), ready: title, wait: 600, steps: [
    tap("main button", "Feature"),
    assert(`document.querySelector('${sheet}')?.textContent.includes('Move to') && ${labels(`${sheet} button.bg-secondary-container`)} === 'Feature'`,
      "the move sheet is not open with Feature checked"),
  ] },
  { name: "task_menu", path: T(3), ready: title, steps: [
    tap("header button[aria-label='More for this task']"),
    assert(`${labels("[role=menu] [role=menuitem]")}.startsWith('Move up,Move down') && ${onScreen("document.querySelector('[role=menu]')")}`,
      "the task's menu is not open on screen"),
  ] },
  { name: "task_picker", path: T(1), ready: title, wait: 600, steps: [
    tap("main button", "High"),
    assert(`document.querySelector('${sheet}')?.textContent.includes('Priority')`, "the priority sheet is not open"),
  ] },
  { name: "task_tags", path: T(9), ready: "main figure img", wait: 600, steps: [
    tap("main button", "Magenta"),
    assert(`[...document.querySelectorAll('${sheet} [role=group][aria-label=Tag] button')].map((b) => b.ariaLabel + (b.ariaPressed === 'true' ? '*' : '')).join() === 'None,Green,Blue,Magenta*,Cyan'`,
      "the tag sheet does not offer the four dots with the task's magenta diamond picked as magenta"),
  ] },
  { name: "cat_menu", path: "/", steps: [
    tap(`${cat(8)} button[aria-label="More for Feature"]`),
    assert(`${labels("[role=menu] [role=menuitem]")} === 'Rename,Add separator,Add sub-category,Select completed,Delete category' && ${onScreen("document.querySelector('[role=menu]')")} && document.querySelector('[role=menu]').getBoundingClientRect().top >= document.querySelector('${cat(8)} button[aria-label="More for Feature"]').getBoundingClientRect().bottom`,
      "the category menu is not open under its button, on screen"),
  ] },
  { name: "top_menu", path: "/", steps: [
    tap("header button[aria-label=More]"),
    assert(`${labels("[role=menu] [role=menuitem]")} === 'Rename project,Delete project,Sync now,Devices' && ${onScreen("document.querySelector('[role=menu]')")}`,
      "the top bar's menu is not open on screen"),
  ] },
  // The typed title is the last thing: the add row saves it when it loses the focus.
  { name: "add_row", path: "/", steps: [
    tap(`${cat(8)} button[aria-label="Add a task to Feature"]`),
    { text: "Offline mode" },
    assert(`document.activeElement.matches('${cat(8)} input[aria-label="New task"]') && document.activeElement.value === 'Offline mode' && document.activeElement.compareDocumentPosition(document.querySelector('${cat(8)} [data-open]')) & Node.DOCUMENT_POSITION_FOLLOWING`,
      "the add row is not first in Feature, focused, with the typed title"),
  ] },
  // A long press let go where it started selects the row; a tap then adds another.
  { name: "select", path: "/", steps: [
    press(`${row(1)} a`),
    tap(`${row(9)} a`),
    assert(`document.querySelector('header h1').textContent.trim() === '2 selected' && ['${row(1)}', '${row(9)}'].every((s) => document.querySelector(s).matches('.bg-secondary-container')) && location.pathname === '/'`,
      "the two rows are not selected"),
  ] },
  { name: "bulk_sheet", path: "/", wait: 600, steps: [
    press(`${row(1)} a`),
    tap("header button[aria-label='Edit selected']"),
    assert(`document.querySelector('${sheet}')?.textContent.includes('Edit 1 selected')`, "the bulk edit sheet is not open on one row"),
  ] },
  { name: "search", path: "/", steps: [
    tap("header button[aria-label=Search]"),
    { text: "the" },
    assert(`document.activeElement.matches('header input[inputmode=search]') && ${shown}.sort().join() === '01,09,10,15,16,17,18,20'`,
      "the search does not narrow the list to the tasks holding “the”"),
  ] },
  { name: "filter_sheet", path: "/", wait: 600, prefs: { filters: { [P]: { statuses: ["in_progress"], priorities: [], categories: [], tags: [] } } }, steps: [
    tap("header button[aria-label=Filter]"),
    assert(`${shown}.join() === '01,03' && ${labels(`${sheet} button.rounded-lg.bg-secondary-container`)} === 'Doing'`,
      "the filter sheet does not show Doing on, over the Doing rows"),
  ] },
  { name: "drag", path: "/", steps: [
    { at: `${row(5)} a`, hold: 600, moves: [[0, -20], [0, -40]] },
    assert(`document.querySelector('[data-block="${A(5)}"]').matches('.fixed') && document.querySelectorAll('[data-cat] div.rounded-2xl[style*=height]').length === 1`,
      "the long press did not lift the row over a gap"),
  ] },
  { name: "devices", path: "/devices", ready: "main li", steps: [
    assert("document.querySelectorAll('main li').length === 1 && /Screenshot\\s*· this device/.test(document.querySelector('main li').textContent)",
      "the paired browser is not listed as this device"),
  ] },
  { name: "devices_pair", path: "/devices", ready: "main li", wait: 600, steps: [
    tap("main button", "Pair a new device"),
    poll(`document.querySelector('${sheet} img[alt="Pairing code"]')?.naturalWidth > 0 && document.querySelector('${sheet} code').textContent.includes('/pair/')`,
      "the pairing sheet shows no code"),
  ] },
  { name: "sign_in", path: "/", anon: true, ready: "main h1", steps: [
    assert("document.querySelector('main h1').textContent === 'Sign in to Ordo' && !document.querySelector('[data-row]') && [...document.querySelectorAll('main button')].some((b) => b.textContent.includes('Scan the code'))",
      "an anonymous browser does not land on the sign-in screen"),
  ] },
  { name: "pair", path: "/pair/sample-token", anon: true, ready: "[role=alert]", steps: [
    assert("document.querySelector('[role=alert]').textContent.includes('used or expired')", "a dead pairing link does not say so"),
  ] },
];
