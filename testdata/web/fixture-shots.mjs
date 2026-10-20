// The dev fixture's shots (testdata/web/seed.json), the driver's default list.
const P = "/p/0rd0-web-fixture";
const T = (n) => `${P}/t/aaaaaaaa-0000-4000-8000-${String(n).padStart(12, "0")}`;
const C = (n) => `cccccccc-0000-4000-8000-${String(n).padStart(12, "0")}`;
const folded = C(24);
// A pointer click (detail 1), unlike element.click(), which reads as a keyboard press.
const tap = (sel) => `document.querySelector('${sel}').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, detail: 1 }))`;

export default [
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
  // Another device deletes the task while its title is typed; the delete's own undo brings it back after.
  { name: "task_gone", path: T(3), wait: 900, steps: [
    "document.querySelector('#title').focus()",
    "fetch(document.querySelector('.title-form').action.replace('task.title', 'task.delete'), { method: 'POST', credentials: 'same-origin', headers: { 'HX-Request': 'true' }, body: new URLSearchParams({ id: document.querySelector('#app').dataset.task }) }).then((r) => { window.undoDelete = JSON.parse(r.headers.get('HX-Trigger'))['ordo:snack'].undo; })",
    "(() => { const f = document.querySelector('#title'); f.value = 'Typed on the phone'; f.dispatchEvent(new Event('change', { bubbles: true })); f.blur(); })()",
  ], after: ["fetch(undoDelete, { method: 'POST', credentials: 'same-origin', headers: { 'HX-Request': 'true' }, body: new URLSearchParams({ back: location.pathname }) })"] },
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
