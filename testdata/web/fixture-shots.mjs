// The dev fixture's shots (testdata/web/seed.json), the driver's default list.
const P = "/p/0rd0-web-fixture";
const T = (n) => `${P}/t/aaaaaaaa-0000-4000-8000-${String(n).padStart(12, "0")}`;
const C = (n) => `cccccccc-0000-4000-8000-${String(n).padStart(12, "0")}`;
const A = (n) => `aaaaaaaa-0000-4000-8000-${String(n).padStart(12, "0")}`;
const folded = C(24);
// A pointer click (detail 1), unlike element.click(), which reads as a keyboard press.
const tap = (sel) => `document.querySelector('${sel}').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, detail: 1 }))`;
const assert = (cond, msg = cond) => `(() => { if (!(${cond})) throw new Error(${JSON.stringify(msg)}); })()`;
const later = (ms, step) => `new Promise((r) => setTimeout(r, ${ms})).then(() => ${step})`;
const post = (action, body) => `fetch(location.pathname.replace(/\\/t\\/.*/, '') + '/do/${action}', { method: 'POST', credentials: 'same-origin', headers: { 'HX-Request': 'true' }, body: new URLSearchParams(${body}) })`;
const rows = (cat) => `[...document.querySelector('#c-${cat} .category-body > .rows').children]`;
const cursorJumps = `#t-${A(9)}`;

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
  // The same conflict, then Back to the list and the task again: the typed title and the box come back.
  { name: "task_conflict_back", path: T(9), wait: 900, steps: [
    "window.rename = (base, title) => fetch(document.querySelector('.title-form').action, { method: 'POST', credentials: 'same-origin', headers: { 'HX-Request': 'true' }, body: new URLSearchParams({ id: document.querySelector('#app').dataset.task, base, title }) }).then((r) => r.status)",
    "window.original = document.querySelector('.title-form [data-base]').value; document.querySelector('#title').focus()",
    "rename(original, 'Renamed on the laptop')",
    "(() => { const f = document.querySelector('#title'); f.value = 'Typed on the phone'; f.dispatchEvent(new Event('change', { bubbles: true })); f.blur(); })()",
    "new Promise((r) => { document.body.addEventListener('htmx:afterSettle', r, { once: true }); document.querySelector('[data-back]').click(); })",
    `new Promise((r) => { document.body.addEventListener('htmx:afterSettle', r, { once: true }); document.querySelector('a.item-body[href="${T(9)}"]').click(); })`,
    assert("document.querySelector('.title-form .conflict') && document.querySelector('#title').value === 'Typed on the phone'", "the typed title or the conflict box did not come back"),
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
  { name: "task_typing", path: T(1), steps: [
    "document.querySelector('#new-subtask').focus()",
    assert("getComputedStyle(document.querySelector('.done-band')).display === 'none'", "the mark done band shows while typing"),
  ] },
  { name: "task_swap", path: T(9), steps: ["htmx.ajax('GET', location.pathname, { target: '#app', select: '#app', swap: 'outerHTML' })"] },
  { name: "add_row", path: P, steps: [
    "document.querySelector('[data-add-fab]').click()",
    assert(`${rows(C(8))}[0].matches('.composer') && document.activeElement === document.querySelector('.composer-input')`, "the add row is not first in the top category and focused"),
    "document.querySelector('.composer-input').value = 'Offline mode'",
  ] },
  // Two Enters in a row: the second waits for the first, so both land above the row in order and the field keeps the focus.
  { name: "add_batch", path: P, steps: [
    tap(`#c-${C(12)} [data-add-to]`),
    "{ const i = document.querySelector('.composer-input'); for (const title of ['One', 'Two']) { i.value = title; i.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })); } }",
    later(600, assert(`${rows(C(12))}.slice(0, 3).map((li) => li.matches('.composer') ? 'row' : li.querySelector('.headline').textContent).join() === 'One,Two,row' && document.activeElement === document.querySelector('.composer-input')`, "One and Two did not land above the focused add row")),
  ], after: [post("task.delete", `${rows(C(12))}.filter((li) => li.matches('.row')).slice(0, 2).map((li) => ['id', li.dataset.id])`)] },
  // Two taps: Doing, then Done in place; a second after, it slides into Completed; Undo brings it back as Doing.
  { name: "status_settle", path: P, steps: [
    `document.getElementById('t-${A(9)}').scrollIntoView({ block: 'center' })`,
    tap(`#t-${A(9)} .status-button`),
    tap(`#t-${A(9)} .status-button`),
    assert(`document.getElementById('t-${A(9)}').matches('.s-completed:not(.completed *)')`, "the second tap did not mark it done in place"),
    later(1800, assert(`document.getElementById('t-${A(9)}').closest('details.completed') && document.querySelector('#c-${C(12)} .completed-toggle').textContent.includes('(3)')`, "the done task did not move into Completed")),
  ], after: [
    "document.querySelector('.snackbar-action').click()",
    assert(`document.getElementById('t-${A(9)}').matches('.s-in_progress:not(.completed *)')`, "Undo did not bring it back as Doing"),
    post("status.set", `{ id: '${A(9)}', value: 'todo' }`),
  ] },
  // A category header scrolled to the bottom edge: the FAB steps aside and a tap on its + reaches it.
  { name: "fab_tucked", path: P, steps: [
    "window.plus = [...document.querySelectorAll('[data-add-to]')].find((b) => b.getBoundingClientRect().top > innerHeight); window.scrollBy(0, plus.getBoundingClientRect().bottom - innerHeight + 10)",
    later(300, assert("(({ left, right, top, bottom }) => document.querySelector('[data-fab]').matches('.tucked') && document.elementFromPoint((left + right) / 2, (top + bottom) / 2).closest('[data-add-to]') === plus)(plus.getBoundingClientRect())", "the FAB still takes taps on a category's +")),
  ] },
  { name: "cat_menu", path: P, steps: [
    tap(`#c-${C(12)} [data-cat-menu]`),
    assert(`(({ top, left, right, bottom }) => document.querySelector('#cat-menu').matches(':popover-open') && Math.abs(top - document.querySelector('#c-${C(12)} [data-cat-menu]').getBoundingClientRect().bottom) < 1 && left >= 0 && right <= innerWidth && bottom <= innerHeight)(document.querySelector('#cat-menu').getBoundingClientRect())`, "the category menu is not open under its button, on screen"),
  ] },
  { name: "separator_below", path: P, wait: 600, steps: [
    "document.querySelector('[data-select-start]').click()",
    `document.querySelector('#t-${A(1)} > .list-item .item-body').click()`,
    "document.querySelector('[data-sheet-open=bulk-sheet]').click()",
    "document.querySelector('[data-separator-below]').click()",
    assert(`!document.querySelector('#bulk-sheet').open && !document.body.classList.contains('selecting') && document.getElementById('t-${A(1)}').nextElementSibling.matches('.separator.editing') && document.activeElement === document.getElementById('t-${A(1)}').nextElementSibling.querySelector('input')`, "no focused label row below the task's block"),
  ], after: [
    "{ const i = document.querySelector('.separator-input'); i.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })); }",
    "document.querySelector('.snackbar-action').click()",
  ] },
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
  { name: "drawer_swipe", path: P, gesture: { point: [12, 400], moves: [[30, 0], [80, 2], [80, 2]] },
    after: [assert("(() => { const d = document.querySelector('#drawer'); const shown = +d.style.getPropertyValue('--shown'); return d.open && d.matches('.tracking') && shown > 0 && shown < 1; })()")] },
  { name: "sheet_swipe", path: P, steps: [
    "document.querySelector('[data-select-start]').click()",
    "document.querySelectorAll('.content .row .item-body')[0].click()",
    "document.querySelector('[data-sheet-open=bulk-sheet]').click()",
  ], gesture: { at: "#bulk-sheet .drag-handle", moves: [[0, 40], [0, 80], [0, 80]] },
  after: [assert("(() => { const d = document.querySelector('#bulk-sheet'); return d.matches('.tracking') && parseFloat(d.style.transform.replace('translateY(', '')) > 0; })()")] },
  { name: "drag_lift", path: P, gesture: { at: "#t-aaaaaaaa-0000-4000-8000-000000000005 > .list-item", hold: 600 },
    after: [assert(`document.querySelector('.drag-clone') && document.querySelector('.drag-slot').id === 't-aaaaaaaa-0000-4000-8000-000000000005' && getSelection().isCollapsed && location.pathname === '${P}'`)] },
  { name: "drag_nest", path: P, gesture: { at: "#t-aaaaaaaa-0000-4000-8000-000000000005 > .list-item", hold: 600, moves: [[20, 0], [30, 4], [20, 0]] },
    after: [assert("document.querySelector('.drag-slot.child').parentElement.closest('.row').id === 't-aaaaaaaa-0000-4000-8000-000000000001'")] },
  { name: "drag_separator", path: P, gesture: { at: "#t-bbbbbbbb-0000-4000-8000-000000000006", hold: 600, moves: [[0, -20], [0, -40], [0, -10]] },
    after: [assert("document.querySelector('.drag-slot.separator:not(.child)') && getSelection().isCollapsed")] },
  { name: "drag_empty", path: P, steps: [
    "document.querySelector('#c-cccccccc-0000-4000-8000-000000000014 .category-body > .rows').replaceChildren()",
    "document.querySelector('#t-aaaaaaaa-0000-4000-8000-000000000009').scrollIntoView({ block: 'center' })",
  ], gesture: { at: "#t-aaaaaaaa-0000-4000-8000-000000000009 > .list-item", hold: 600, moves: [[0, 16], [0, 16]] },
  after: [assert("document.body.matches('.dragging') && document.querySelector('.drag-slot').id === 't-aaaaaaaa-0000-4000-8000-000000000009'")] },
  { name: "drag_select", path: P, gesture: { at: `${cursorJumps} > .list-item`, hold: 600, release: true },
    after: [assert(`!document.querySelector('[data-selectbar]').hidden && document.querySelector('[data-select-count]').textContent === '1' && document.querySelector('${cursorJumps}').matches('.selected') && location.pathname === '${P}'`)] },
  // Undo puts the row back for the next color scheme's pass.
  { name: "drag_drop", path: P, steps: [`document.querySelector('${cursorJumps}').scrollIntoView({ block: 'center' })`], gesture: { at: `${cursorJumps} > .list-item`, hold: 600, moves: [[0, -20], [0, -60], [0, -70]], release: true }, wait: 900, after: [
    assert(`document.querySelector('.snackbar-text').textContent === 'Moved' && document.querySelector('${cursorJumps}').closest('.category').id === 'c-${C(8)}' && location.pathname === '${P}'`),
    "document.querySelector('.snackbar-action').click()",
    assert(`document.querySelector('${cursorJumps}').closest('.category').id === 'c-${C(12)}'`),
  ] },
  { name: "snackbar", path: P, steps: ["document.querySelector('.status-button').click()"], wait: 900,
    after: ["document.querySelector('.snackbar-action').click()"] },
  { name: "devices", path: "/devices" },
  { name: "devices_pair", path: "/devices", steps: ["document.querySelector('form[action=\"/devices/pair\"] button').click()"], wait: 900 },
  { name: "login", path: "/login", anon: true },
  { name: "scan_nocamera", path: "/login", anon: true, steps: ["document.querySelector('[data-scan]').click()"], wait: 900 },
  // The camera prompt refused: the sheet closes, the note says so, the field takes the pasted link.
  { name: "scan_blocked", path: "/login", anon: true, wait: 900, steps: [
    "navigator.mediaDevices.getUserMedia = () => Promise.reject(new DOMException('Permission denied', 'NotAllowedError')); document.querySelector('[data-scan]').click()",
    // The sheet slides away for up to 400 ms before it closes and hands the field focus.
    later(500, assert("!document.querySelector('#scan-sheet').open && /blocked/.test(document.querySelector('[data-scan-note]').textContent) && document.activeElement === document.querySelector('[data-pair-token]')", "the sheet stayed, the note is wrong or the field is not focused")),
  ] },
  { name: "pair", path: "/pair/sample-token", anon: true, steps: [
    assert("!document.querySelector('[data-browser-hint]').hidden", "the browser tab hint is hidden"),
  ] },
];
