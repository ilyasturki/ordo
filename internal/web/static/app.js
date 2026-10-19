"use strict";

(() => {
  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));
  const app = () => $("#app");

  const selected = new Set();
  let selecting = false;
  let dragging = false;
  let staleStamp = "";
  let snackTimer = 0;
  let lingering = false;

  function snack(msg, undo) {
    const bar = $("#snackbar");
    if (!bar || !msg) return;
    $(".snackbar-text", bar).textContent = msg;
    const action = $(".snackbar-action", bar);
    action.hidden = !undo;
    action.textContent = undo === "reload" ? "Reload" : "Undo";
    action.onclick = () => {
      bar.hidden = true;
      if (undo === "reload") refresh();
      else htmx.ajax("POST", undo, { target: "#app", select: "#app", swap: "outerHTML", values: { back: location.pathname + location.search } });
    };
    bar.hidden = false;
    bar.style.animation = "none";
    void bar.offsetWidth;
    bar.style.animation = "";
    clearTimeout(snackTimer);
    snackTimer = setTimeout(() => { bar.hidden = true; }, undo ? 6000 : 3000);
  }

  function flash(id) {
    const el = document.getElementById(id);
    if (!el) return;
    el.scrollIntoView({ block: "nearest", behavior: "smooth" });
    el.classList.remove("flash");
    void el.offsetWidth;
    el.classList.add("flash");
  }

  function busy() {
    const a = document.activeElement;
    return dragging || lingering || swipe || $("dialog[open]") || (a && a.matches("input:not([type=hidden]), textarea, select"));
  }

  function refresh() {
    staleStamp = "";
    htmx.ajax("GET", location.pathname + location.search, {
      target: "#app", select: "#app", swap: "outerHTML", headers: { "X-Ordo-Refresh": "1" },
    });
  }

  function refreshWhenIdle() {
    if (staleStamp && !busy()) refresh();
  }

  let source = null;
  function listen() {
    const root = app();
    const pid = root && root.dataset.project;
    if (source && source.pid === pid) return;
    if (source) source.close();
    source = null;
    if (!pid) return;
    source = new EventSource("/p/" + pid + "/events");
    source.pid = pid;
    source.addEventListener("change", (e) => {
      const cur = app();
      if (!cur || e.data === cur.dataset.stamp) return;
      staleStamp = e.data;
      refreshWhenIdle();
    });
  }

  function updateBars() {
    const bar = $("[data-appbar]");
    if (bar) bar.classList.toggle("scrolled", window.scrollY > 4);
  }

  let lastY = 0;
  window.addEventListener("scroll", () => {
    updateBars();
    const fab = $("[data-fab]");
    if (fab) {
      const y = window.scrollY;
      if (Math.abs(y - lastY) > 8) fab.classList.toggle("collapsed", y > lastY && y > 80);
      lastY = y;
    }
  }, { passive: true });

  function syncSelection() {
    const root = app();
    if (!root || root.dataset.page !== "project") return;
    document.body.classList.toggle("selecting", selecting);
    const appbar = $("[data-appbar]");
    const selbar = $("[data-selectbar]");
    if (appbar) appbar.hidden = selecting;
    if (selbar) selbar.hidden = !selecting;
    for (const row of $$(".row")) {
      if (!document.getElementById(row.id)) continue;
      const on = selected.has(row.dataset.id);
      row.classList.toggle("selected", on);
      row.setAttribute("aria-selected", on ? "true" : "false");
    }
    for (const id of Array.from(selected)) if (!document.getElementById("t-" + id)) selected.delete(id);
    for (const el of $$("[data-select-count]")) el.textContent = selected.size;
    const form = $("[data-bulk-form]");
    if (form) {
      $$("input[name=id]", form).forEach((i) => i.remove());
      for (const id of selected) {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = "id";
        input.value = id;
        form.append(input);
      }
    }
    for (const b of $$("[form=bulk]")) b.disabled = selected.size === 0;
    const below = $("[data-separator-below]");
    if (below) {
      const only = selected.size === 1 && document.getElementById("t-" + [...selected][0]);
      below.disabled = !only || !!only.closest(".completed");
    }
    for (const s of sortables) s.option("disabled", selecting);
  }

  function startSelecting(id) {
    selecting = true;
    if (id) selected.add(id);
    syncSelection();
  }

  function stopSelecting() {
    selecting = false;
    selected.clear();
    syncSelection();
  }

  function toggleSelected(id) {
    if (selected.has(id)) selected.delete(id);
    else selected.add(id);
    if (selected.size === 0) stopSelecting();
    else syncSelection();
  }

  // send posts an edit outside htmx; with want "row" the answer is that row's markup, "" when refused.
  async function send(url, values, want) {
    let res;
    try {
      res = await fetch(url, {
        method: "POST",
        body: new URLSearchParams(values),
        credentials: "same-origin",
        headers: { "HX-Request": "true", "X-Ordo-Want": want || "", "X-Ordo-Page": app() ? app().dataset.stamp : "" },
      });
    } catch {
      snack("Offline: the change didn't reach Ordo", "reload");
      return null;
    }
    if (!res.ok) {
      snack(res.status === 401 ? "Signed out" : "That didn't work. Check the connection.");
      return null;
    }
    const trigger = res.headers.get("HX-Trigger");
    const sn = trigger && JSON.parse(trigger)["ordo:snack"];
    if (sn) snack(sn.msg, sn.undo);
    adoptStamp(res.headers.get("X-Ordo-Stamp"));
    return want ? res.text() : "";
  }

  function rowFrom(html) {
    const tpl = document.createElement("template");
    tpl.innerHTML = html.trim();
    const row = tpl.content.firstElementChild;
    if (row) htmx.process(row);
    return row;
  }

  function unfold(section) {
    if (!section.classList.contains("folded")) return;
    section.classList.remove("folded");
    $("[data-fold]", section).setAttribute("aria-expanded", "true");
    fetch("/fold/" + encodeURIComponent(section.dataset.cat), { method: "POST", credentials: "same-origin" });
  }

  function topCategory() {
    const bar = $("[data-appbar]");
    const top = bar ? bar.getBoundingClientRect().bottom : 0;
    const cats = $$(".content .category");
    return cats.find((c) => c.getBoundingClientRect().bottom > top + 48) || cats[0];
  }

  // openComposer puts the inline add row first in the category and focuses it within the tap, so the keyboard opens at once.
  function openComposer(section) {
    const tpl = $("#composer");
    if (!tpl || !section) return;
    unfold(section);
    const li = tpl.content.firstElementChild.cloneNode(true);
    li.dataset.cat = section.dataset.cat;
    $(".category-body > .rows", section).prepend(li);
    li.scrollIntoView({ block: "nearest" });
    $("input", li).focus({ preventScroll: true });
    if (window.CloseWatcher) {
      try {
        li.watcher = new CloseWatcher();
        li.watcher.onclose = () => closeComposer(li);
      } catch {}
    }
  }

  function closeComposer(li) {
    if (li.closed) return;
    li.closed = true;
    if (li.watcher) li.watcher.destroy();
    const input = $("input", li);
    if (document.activeElement === input) input.blur();
    li.hidden = true;
    (li.pending || Promise.resolve()).then(() => li.remove());
  }

  // submitComposer adds the typed task above the row, which then stands where the next one lands.
  function submitComposer(li, close) {
    const input = $("input", li);
    const title = input.value.trim();
    if (title) {
      input.value = "";
      li.pending = (li.pending || Promise.resolve()).then(async () => {
        const values = { title, cat: li.dataset.cat, back: location.pathname };
        if (li.dataset.after) values.id = li.dataset.after;
        const html = await send($("form", li).action, values, "row");
        const row = html && rowFrom(html);
        if (!row) {
          if (!input.value) input.value = title;
          return;
        }
        li.before(row);
        li.dataset.after = row.dataset.id;
        const empty = $(".category-body > .empty", li.closest(".category"));
        if (empty) empty.remove();
        row.classList.add("flash");
      });
    }
    if (close) closeComposer(li);
  }

  const doURL = (action) => "/p/" + app().dataset.project + "/do/" + action;

  const cycle = ["todo", "in_progress", "completed", "cancelled"];
  const isFinished = (status) => status === "completed" || status === "cancelled";
  const statusOf = (row) => cycle.find((s) => row.classList.contains("s-" + s)) || "todo";
  const touched = new Set();
  let lingerTimer = 0;
  let taps = 0;
  let statusQueue = Promise.resolve();
  let afterRefresh = null;

  function showStatus(row, status) {
    const button = $(":scope > .list-item .status-button", row);
    const icon = $("#status-icons").content.querySelector(`[data-status="${status}"]`);
    for (const s of cycle) {
      row.classList.toggle("s-" + s, s === status);
      button.classList.toggle("s-" + s, s === status);
    }
    button.replaceChildren(icon.firstElementChild.cloneNode(true));
    button.setAttribute("aria-label", icon.dataset.label + ", change status");
  }

  // stepStatus moves the row along the TUI's cycle at once; the list settles a second after the last tap.
  function stepStatus(button) {
    const row = button.closest(".row");
    const next = cycle[(cycle.indexOf(statusOf(row)) + 1) % cycle.length];
    showStatus(row, next);
    touched.add(row.closest(".category-body > .rows > li, .completed-rows > li"));
    const values = { id: row.dataset.id, value: next, back: location.pathname };
    statusQueue = statusQueue.then(() => send(doURL("status.set"), values));
    taps++;
    lingering = true;
    clearTimeout(lingerTimer);
    lingerTimer = setTimeout(settleStatuses, 1000);
  }

  // settleStatuses slides out each block that now belongs on the other side of its Completed fold, then redraws.
  async function settleStatuses() {
    const tap = taps;
    await statusQueue;
    if (tap !== taps) return;
    const moving = [...touched].filter((block) => {
      if (!block || !block.isConnected) return false;
      const done = [block, ...$$(".row", block)].every((r) => isFinished(statusOf(r)));
      return done !== !!block.closest(".completed");
    });
    touched.clear();
    for (const block of moving) {
      block.style.height = block.offsetHeight + "px";
      block.classList.add("leaving");
    }
    void document.body.offsetHeight;
    for (const block of moving) block.style.height = "0px";
    if (moving.length) await new Promise((r) => setTimeout(r, 250));
    if (tap !== taps) return;
    const ids = moving.map((block) => block.dataset.id);
    afterRefresh = () => {
      for (const id of ids) {
        const row = document.getElementById("t-" + id);
        const shut = row && row.closest("details.completed:not([open])");
        const el = shut ? $(".completed-toggle", shut) : row;
        if (!el) continue;
        el.classList.remove("flash");
        void el.offsetWidth;
        el.classList.add("flash");
      }
    };
    lingering = false;
    if (app() && app().dataset.page === "project") refresh();
  }

  // openSeparatorEditor puts a label field in place and focuses it within the tap, like the add row.
  function openSeparatorEditor(place, base) {
    const li = $("#separator-editor").content.firstElementChild.cloneNode(true);
    place(li);
    const input = $("input", li);
    input.value = base;
    li.dataset.base = base;
    li.scrollIntoView({ block: "nearest" });
    input.focus({ preventScroll: true });
    if (window.CloseWatcher) {
      try {
        li.watcher = new CloseWatcher();
        li.watcher.onclose = () => finishSeparator(li, false);
      } catch {}
    }
    return li;
  }

  // newSeparator adds a bare separator first in the category, or below the block of row, then opens its label.
  function newSeparator(section, row) {
    unfold(section);
    const li = openSeparatorEditor((el) => {
      if (row) row.closest(".category-body > .rows > li").after(el);
      else $(".category-body > .rows", section).prepend(el);
    }, "");
    const values = { cat: section.dataset.cat, back: location.pathname };
    if (row) values.id = row.dataset.id;
    li.pending = send(doURL("separator.add"), values, "row").then((html) => {
      li.row = html && rowFrom(html);
      if (li.row) li.dataset.id = li.row.dataset.id;
    });
  }

  function relabelSeparator(sep) {
    const li = openSeparatorEditor((el) => {
      sep.hidden = true;
      sep.after(el);
    }, $(".separator-label", sep).textContent);
    li.dataset.id = sep.dataset.id;
    li.row = sep;
  }

  function finishSeparator(li, save) {
    if (li.closed) return;
    li.closed = true;
    if (li.watcher) li.watcher.destroy();
    const input = $("input", li);
    if (document.activeElement === input) input.blur();
    const title = input.value.trim();
    (li.pending || Promise.resolve()).then(() => {
      if (!li.isConnected) return;
      if (save && li.row && title !== li.dataset.base) {
        htmx.ajax("POST", doURL("task.title"), { target: "#app", select: "#app", swap: "outerHTML", values: { id: li.dataset.id, title, base: li.dataset.base, back: location.pathname } });
        return;
      }
      if (!li.row) {
        li.remove();
        return;
      }
      li.row.hidden = false;
      li.replaceWith(li.row);
    });
  }

  // reopenBlocked: the press that light-dismissed the menu was on its own ⋮, so that tap only closes it.
  let reopenBlocked = false;
  function openMenu(button, byKeyboard) {
    const menu = $("#cat-menu");
    if (!menu || !menu.showPopover || reopenBlocked) return;
    const section = button.closest(".category");
    menu.opener = button;
    menu.section = section;
    const folded = section.classList.contains("folded");
    $("[data-menu=fold]", menu).hidden = folded;
    $("[data-menu=unfold]", menu).hidden = !folded;
    $("[data-menu=select]", menu).disabled = !$(".completed-rows .row", section);
    menu.showPopover();
    const r = button.getBoundingClientRect();
    const w = menu.offsetWidth;
    const h = menu.offsetHeight;
    menu.style.top = (r.bottom + h > window.innerHeight - 8 ? Math.max(8, r.top - h) : r.bottom) + "px";
    menu.style.left = Math.max(8, Math.min(r.right - w, window.innerWidth - w - 8)) + "px";
    const first = $(".menu-item:not([hidden]):not(:disabled)", menu);
    if (first && byKeyboard) first.focus();
  }

  function menuAction(item) {
    const menu = $("#cat-menu");
    const section = menu.section;
    menu.hidePopover();
    if (!section || !section.isConnected) return;
    switch (item.dataset.menu) {
      case "separator":
        newSeparator(section, null);
        break;
      case "fold":
      case "unfold":
        $("[data-fold]", section).click();
        break;
      case "select":
        selectCompleted(section);
        break;
    }
  }

  function selectCompleted(section) {
    const fold = $(".completed", section);
    if (!fold) return;
    if (!fold.open) {
      fold.open = true;
      fetch("/fold/" + encodeURIComponent($("[data-completed-fold]", fold).dataset.completedFold), { method: "POST", credentials: "same-origin" });
    }
    selecting = true;
    for (const row of $$(".completed-rows .row", fold)) selected.add(row.dataset.id);
    syncSelection();
  }

  const exitOf = (d) => d.getAnimations().find((a) => a.transitionProperty === "transform" || a.transitionProperty === "opacity");

  // A closed dialog leaves the top layer at once and Firefox has no `overlay` to keep it there, so it animates out while still open.
  function dismiss(d) {
    if (!d || !d.open) return 0;
    if (!d.classList.contains("closing")) {
      if (d.contains(document.activeElement)) document.activeElement.blur();
      d.classList.remove("tracking");
      unstyle(d);
      d.classList.add("closing");
      const done = () => {
        if (!d.classList.contains("closing")) return;
        d.classList.remove("closing");
        d.close();
      };
      const exit = exitOf(d);
      if (!exit) {
        done();
        return 0;
      }
      exit.finished.then(done, done);
      setTimeout(done, 400);
    }
    const exit = exitOf(d);
    return exit ? Math.max(0, exit.effect.getComputedTiming().endTime - (exit.currentTime || 0)) : 0;
  }

  document.addEventListener("cancel", (e) => {
    if (!(e.target instanceof HTMLDialogElement) || !e.cancelable) return;
    e.preventDefault();
    dismiss(e.target);
  }, true);

  let swipe = null;
  let swallowUntil = 0;

  function swipeFrom(t, x, y) {
    const d = t.closest("dialog.drawer[open], dialog.sheet[open]");
    if (d) {
      if (d.classList.contains("closing")) return null;
      if (d.matches(".drawer")) return { d, kind: "drawer" };
      if (y < d.getBoundingClientRect().top || t.closest("input, textarea, select")) return null;
      const scroller = t.closest(".sheet-scroll");
      return scroller && scroller.scrollTop > 0 ? null : { d, kind: "sheet" };
    }
    const bar = t.closest("#snackbar");
    if (bar) return { d: bar, kind: "snack" };
    const drawer = $("#drawer");
    if (drawer && !drawer.open && x <= 40 && !selecting && !t.closest("input, textarea, select") && !scrollsSideways(t)) return { d: drawer, kind: "edge" };
    return null;
  }

  function scrollsSideways(el) {
    for (; el && el !== document.body; el = el.parentElement) {
      if (el.scrollWidth > el.clientWidth && /auto|scroll/.test(getComputedStyle(el).overflowX)) return true;
    }
    return false;
  }

  function beginSwipe(s) {
    const d = s.d;
    s.axis = s.kind === "sheet" ? "y" : "x";
    d.classList.add("tracking");
    if (s.kind === "edge") {
      d.style.transform = "translateX(-100%)";
      d.style.setProperty("--shown", "0");
      d.showModal();
    }
    s.size = s.axis === "y" ? d.offsetHeight : d.offsetWidth;
  }

  function followSwipe(s, dx, dy) {
    const now = performance.now();
    s.samples.push([now, s.axis === "y" ? dy : dx]);
    while (now - s.samples[0][0] > 100) s.samples.shift();
    if (s.kind === "sheet") s.off = Math.max(0, dy);
    else if (s.kind === "snack") s.off = dx;
    else s.off = Math.min(0, Math.max(-s.size, dx - (s.kind === "edge" ? s.size : 0)));
    s.d.style.transform = s.axis === "y" ? `translateY(${s.off}px)` : `translateX(${s.off}px)`;
    if (s.kind !== "snack") s.d.style.setProperty("--shown", String(1 - Math.abs(s.off) / s.size));
  }

  // px/ms over the last 100ms; a finger resting before the release sends no moves, so it reads as still.
  function velocity(samples) {
    const now = performance.now();
    const recent = samples.filter(([t]) => now - t <= 100);
    if (recent.length < 2) return 0;
    const [t0, p0] = recent[0];
    const p1 = recent[recent.length - 1][1];
    return (p1 - p0) / Math.max(now - t0, 16);
  }

  function unstyle(el) {
    el.style.removeProperty("transform");
    el.style.removeProperty("--shown");
    if (!el.style.length) el.removeAttribute("style");
  }

  function settleSwipe(s) {
    const v = velocity(s.samples);
    const d = s.d;
    if (s.kind === "snack") {
      d.classList.remove("tracking");
      unstyle(d);
      const from = { transform: `translateX(${s.off}px)` };
      if (Math.abs(v) > 0.5 || Math.abs(s.off) > d.offsetWidth * 0.4) {
        const to = { transform: `translateX(${Math.sign(s.off) * d.offsetWidth}px)`, opacity: 0 };
        const a = d.animate([from, to], { duration: 150, easing: "cubic-bezier(0.3, 0, 0.8, 0.15)", fill: "forwards" });
        a.finished.then(() => { d.hidden = true; a.cancel(); }, () => {});
      } else {
        d.animate([from, { transform: "none" }], { duration: 200, easing: "cubic-bezier(0.2, 0, 0, 1)" });
      }
      return;
    }
    const shut = s.kind === "sheet" ? v > 0.5 || (v > -0.5 && s.off > s.size / 4) : v < -0.3 || (v < 0.3 && s.off < -s.size / 2);
    if (shut) {
      dismiss(d);
      return;
    }
    d.classList.remove("tracking");
    unstyle(d);
  }

  document.addEventListener("touchstart", (e) => {
    if (e.touches.length > 1) {
      if (swipe && swipe.axis) settleSwipe(swipe);
      swipe = null;
      return;
    }
    const p = e.touches[0];
    const s = swipeFrom(e.target, p.clientX, p.clientY);
    swipe = s && { ...s, id: p.identifier, x: p.clientX, y: p.clientY, axis: null, samples: [] };
  }, { passive: true });

  document.addEventListener("touchmove", (e) => {
    if (!swipe) return;
    const p = Array.from(e.changedTouches).find((q) => q.identifier === swipe.id);
    if (!p) return;
    const dx = p.clientX - swipe.x;
    const dy = p.clientY - swipe.y;
    if (!swipe.axis) {
      const ax = Math.abs(dx);
      const ay = Math.abs(dy);
      // A sheet decides on the first move: once the browser starts scrolling, it stops letting the page cancel it.
      if (ax + ay < (swipe.kind === "sheet" ? 3 : 10)) return;
      const ok = swipe.kind === "sheet" ? dy > 0 && ay >= ax : ax > ay * (swipe.kind === "edge" ? 1.5 : 1) && (swipe.kind !== "edge" || dx > 0);
      if (!ok) {
        swipe = null;
        return;
      }
      beginSwipe(swipe);
    }
    if (e.cancelable) e.preventDefault();
    followSwipe(swipe, dx, dy);
  }, { passive: false });

  function endSwipe(e) {
    if (!swipe || !Array.from(e.changedTouches).some((q) => q.identifier === swipe.id)) return;
    if (swipe.axis) {
      settleSwipe(swipe);
      swallowUntil = performance.now() + 400;
    }
    swipe = null;
  }
  document.addEventListener("touchend", endSwipe);
  document.addEventListener("touchcancel", endSwipe);

  function grow(el) {
    el.style.height = "auto";
    el.style.height = el.scrollHeight + "px";
  }

  let sortables = [];
  function setupSortable(root) {
    sortables = [];
    if (!window.Sortable || root.dataset.page !== "project") return;
    for (const list of $$(".content .rows", root)) {
      if (list.closest(".completed")) continue;
      sortables.push(Sortable.create(list, {
        group: "rows",
        draggable: ".row, .separator",
        filter: ".status-button",
        preventOnFilter: false,
        delay: 300,
        delayOnTouchOnly: true,
        touchStartThreshold: 6,
        animation: 150,
        fallbackOnBody: true,
        swapThreshold: 0.6,
        invertSwap: true,
        onChoose: () => { if (navigator.vibrate) navigator.vibrate(10); },
        onStart: () => { dragging = true; document.body.classList.add("dragging"); },
        onEnd: (e) => {
          dragging = false;
          document.body.classList.remove("dragging");
          const id = e.item.dataset.id;
          if (e.from === e.to && e.oldIndex === e.newIndex) {
            refreshWhenIdle();
            return;
          }
          let next = e.item.nextElementSibling;
          while (next && !next.dataset.id) next = next.nextElementSibling;
          const cat = e.to.closest("[data-cat]").dataset.cat;
          htmx.ajax("POST", "/p/" + app().dataset.project + "/do/task.place", {
            target: "#app", select: "#app", swap: "outerHTML",
            values: { id, cat, parent: e.to.dataset.parent || "", before: next ? next.dataset.id : "", back: location.pathname + location.search },
          });
        },
      }));
    }
  }

  let pressTimer = 0;
  let pressOrigin = null;
  document.addEventListener("pointerdown", (e) => {
    const row = e.target.closest(".content .row");
    if (!row || e.target.closest(".status-button") || e.pointerType === "mouse") return;
    pressOrigin = { x: e.clientX, y: e.clientY, id: row.dataset.id };
    clearTimeout(pressTimer);
    pressTimer = setTimeout(() => {
      if (pressOrigin && !dragging) {
        pressOrigin.fired = true;
        if (!selecting) startSelecting(pressOrigin.id);
      }
    }, 550);
  });
  document.addEventListener("pointermove", (e) => {
    if (pressOrigin && Math.hypot(e.clientX - pressOrigin.x, e.clientY - pressOrigin.y) > 8) clearTimeout(pressTimer);
  });
  document.addEventListener("pointerup", () => clearTimeout(pressTimer));
  document.addEventListener("pointercancel", () => clearTimeout(pressTimer));
  let lastPointer = "mouse";
  document.addEventListener("pointerdown", (e) => { lastPointer = e.pointerType; }, true);
  document.addEventListener("contextmenu", (e) => {
    if (e.target.closest(".content .row, .content .separator") || (lastPointer !== "mouse" && e.target.closest(".drawer-item"))) e.preventDefault();
  });

  document.addEventListener("click", (e) => {
    const t = e.target;
    if (performance.now() < swallowUntil) {
      swallowUntil = 0;
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    if (pressOrigin && pressOrigin.fired) {
      pressOrigin = null;
      if (t.closest(".content .row")) {
        e.preventDefault();
        e.stopPropagation();
        return;
      }
    }
    pressOrigin = null;
    const statusButton = t.closest(".content .status-button");
    if (statusButton && app().dataset.page === "project") {
      e.preventDefault();
      stepStatus(statusButton);
      return;
    }
    const fold = t.closest("[data-fold]");
    if (fold) {
      const section = fold.closest(".category");
      const folded = section.classList.toggle("folded");
      fold.setAttribute("aria-expanded", folded ? "false" : "true");
      fetch("/fold/" + encodeURIComponent(fold.dataset.fold), { method: "POST", credentials: "same-origin" });
      return;
    }
    const completedFold = t.closest("[data-completed-fold]");
    if (completedFold) {
      fetch("/fold/" + encodeURIComponent(completedFold.dataset.completedFold), { method: "POST", credentials: "same-origin" });
      return;
    }
    const add = t.closest("[data-add-to]");
    if (add) {
      openComposer(add.closest(".category"));
      return;
    }
    if (t.closest("[data-add-fab]")) {
      openComposer(topCategory());
      return;
    }
    const menuButton = t.closest("[data-cat-menu]");
    if (menuButton) {
      openMenu(menuButton, e.detail === 0);
      return;
    }
    const menuItem = t.closest("[data-menu]");
    if (menuItem) {
      menuAction(menuItem);
      return;
    }
    if (t.closest("[data-separator-below]")) {
      const row = document.getElementById("t-" + [...selected][0]);
      t.closest("dialog").close();
      stopSelecting();
      if (row) newSeparator(row.closest(".category"), row);
      return;
    }
    if (t.closest("[data-select-start]")) {
      startSelecting();
      return;
    }
    if (t.closest("[data-select-end]")) {
      stopSelecting();
      return;
    }
    const nav = t.closest(".drawer-item[href]");
    if (nav) {
      if (nav.matches("[aria-current]")) {
        e.preventDefault();
        e.stopPropagation();
      }
      dismiss(nav.closest("dialog"));
      return;
    }
    const open = t.closest("[data-sheet-open]");
    if (open) {
      const d = document.getElementById(open.dataset.sheetOpen);
      if (d) d.showModal();
      return;
    }
    const pick = t.closest("[data-conflict]");
    if (pick) {
      resolveConflict(pick.closest("form"), pick.dataset.conflict);
      return;
    }
    if (t.closest("[data-sheet-close]")) {
      dismiss(t.closest("dialog"));
      return;
    }
    if (t instanceof HTMLDialogElement && t.open) {
      const r = t.getBoundingClientRect();
      if (e.clientY < r.top || e.clientY > r.bottom || e.clientX < r.left || e.clientX > r.right) {
        dismiss(t);
      }
      return;
    }
    const sep = t.closest(".content .separator[data-sep]");
    if (sep && !selecting) {
      relabelSeparator(sep);
      return;
    }
    if (selecting) {
      const row = t.closest(".content .row");
      if (row && !t.closest(".status-button")) {
        e.preventDefault();
        e.stopPropagation();
        toggleSelected(row.dataset.id);
      }
    }
  }, true);

  document.addEventListener("keydown", (e) => {
    const t = e.target;
    if (e.key === "Enter" && t.matches("[data-single-line]")) {
      e.preventDefault();
      t.blur();
    }
    if (e.key === "Escape" && selecting && !$("dialog[open]")) stopSelecting();
    const composer = t.closest(".composer");
    if (composer && e.key === "Enter" && !e.isComposing) {
      e.preventDefault();
      submitComposer(composer, false);
    }
    if (composer && e.key === "Escape") {
      e.preventDefault();
      closeComposer(composer);
    }
    const label = t.closest(".separator.editing");
    if (label && (e.key === "Enter" && !e.isComposing || e.key === "Escape")) {
      e.preventDefault();
      finishSeparator(label, e.key === "Enter");
    }
  });

  // A tap on an add button leaves the focus in the open composer, so the keyboard stays up between the two rows.
  document.addEventListener("mousedown", (e) => {
    if (e.target.closest("[data-add-to], [data-add-fab]") && document.activeElement && document.activeElement.closest(".composer")) e.preventDefault();
  });

  document.addEventListener("focusout", (e) => {
    const composer = e.target.closest && e.target.closest(".composer");
    if (composer && !composer.closed && !composer.contains(e.relatedTarget) && document.hasFocus()) submitComposer(composer, true);
    const label = e.target.closest && e.target.closest(".separator.editing");
    if (label && !label.closed && document.hasFocus()) finishSeparator(label, true);
  });

  document.addEventListener("pointerdown", (e) => {
    const menu = $("#cat-menu");
    reopenBlocked = !!menu && menu.matches(":popover-open") && e.target.closest("[data-cat-menu]") === menu.opener;
  }, true);

  document.addEventListener("input", (e) => {
    const t = e.target;
    if (t.matches("[data-autogrow]")) grow(t);
    const form = t.closest("[data-autosave]");
    if (form && t.matches("textarea[name=text]")) {
      clearTimeout(form.saveTimer);
      form.saveTimer = setTimeout(() => save(form), 1200);
    }
  });

  // Text typed into a title or details field, by form, until the server has it:
  // { mine, base } while unsent, { mine, theirs } once refused as stale.
  const drafts = new Map(readDrafts());

  // Kept for the tab, so a reload (the back button is one) still has them.
  function readDrafts() {
    try {
      return JSON.parse(sessionStorage.getItem("ordo:drafts")) || [];
    } catch {
      return [];
    }
  }

  function setDraft(key, draft) {
    if (draft) drafts.set(key, draft);
    else drafts.delete(key);
    try {
      sessionStorage.setItem("ordo:drafts", JSON.stringify([...drafts]));
    } catch {}
  }

  function draftKey(form) {
    return form.getAttribute("action") + "#" + $("input[name=id]", form).value;
  }

  function liveForm(key) {
    return $$("[data-autosave]").find((f) => draftKey(f) === key);
  }

  function fieldOf(form) {
    return $("textarea[name=title], textarea[name=text]", form);
  }

  function setImages(form, images) {
    const field = fieldOf(form);
    $$("input[name=image]", form).forEach((i) => i.remove());
    for (const line of images || []) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = "image";
      input.value = line;
      field.closest("label").before(input);
    }
  }

  function showConflict(form) {
    const draft = drafts.get(draftKey(form));
    const field = fieldOf(form);
    if (field.value !== draft.mine) {
      field.value = draft.mine;
      grow(field);
    }
    form.savedValue = field.value;
    let box = $(".conflict", form);
    if (!box) {
      box = document.createElement("div");
      box.className = "conflict";
      box.setAttribute("role", "alert");
      const text = document.createElement("p");
      text.className = "conflict-text";
      text.textContent = "Changed on another device while you typed. Theirs:";
      const quote = document.createElement("p");
      quote.className = "conflict-theirs";
      const actions = document.createElement("div");
      actions.className = "conflict-actions";
      for (const [pick, label] of [["theirs", "Use theirs"], ["mine", "Keep mine"]]) {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "button text";
        button.dataset.conflict = pick;
        button.textContent = label;
        actions.append(button);
      }
      box.append(text, quote, actions);
      form.append(box);
    }
    $(".conflict-theirs", box).textContent = draft.theirs.text.trim() || "(empty)";
  }

  function resolveConflict(form, pick) {
    const key = draftKey(form);
    const draft = drafts.get(key);
    if (!draft || !draft.theirs) return;
    setDraft(key);
    $(".conflict", form).remove();
    $("[data-base]", form).value = draft.theirs.base;
    if (form.matches(".details-form")) setImages(form, draft.theirs.images);
    const field = fieldOf(form);
    if (pick === "mine") {
      form.savedValue = null;
      save(form);
      return;
    }
    field.value = draft.theirs.text;
    grow(field);
    form.savedValue = field.value;
    refresh();
  }

  // A swap of #app renders the fields from the file: what was typed comes back over it.
  function stashDrafts() {
    for (const form of $$("[data-autosave]")) {
      const field = fieldOf(form);
      if (!field || form.savedValue === undefined) continue;
      const key = draftKey(form);
      const draft = drafts.get(key);
      if (draft) setDraft(key, { ...draft, mine: field.value });
      else if (form.savedValue != null && field.value !== form.savedValue) setDraft(key, { mine: field.value, base: $("[data-base]", form).value });
    }
  }

  // Runs as soon as a swap lands, so a second swap never stashes the file's text as typed.
  function adoptForms(root) {
    for (const form of $$("[data-autosave]", root)) {
      if (form.savedValue !== undefined) continue;
      const field = $("textarea", form);
      form.savedValue = field ? field.value : "";
      restoreDraft(form);
    }
  }

  // Mirrors editTitle's TrimSpace and joinDescription's trailing-newline trim.
  function asStored(field, text) {
    return field.name === "title" ? text.trim() : text.replace(/\n+$/, "");
  }

  function restoreDraft(form) {
    const key = draftKey(form);
    const draft = drafts.get(key);
    const field = fieldOf(form);
    if (!draft || !field) return;
    if (asStored(field, draft.mine) === asStored(field, field.value)) {
      setDraft(key);
      return;
    }
    if (draft.theirs) {
      setDraft(key, { mine: draft.mine, theirs: { base: $("[data-base]", form).value, text: field.value, images: $$("input[name=image]", form).map((i) => i.value) } });
      showConflict(form);
      return;
    }
    // Still checked against the text it was typed over, so a newer one on disk is refused, not overwritten.
    setDraft(key);
    $("[data-base]", form).value = draft.base;
    field.value = draft.mine;
    grow(field);
    form.saveTimer = setTimeout(() => save(form), 1200);
  }

  function save(form) {
    clearTimeout(form.saveTimer);
    const field = fieldOf(form);
    if (!field || field.value === form.savedValue) return;
    const key = draftKey(form);
    const draft = drafts.get(key);
    if (draft && draft.theirs) {
      setDraft(key, { ...draft, mine: field.value });
      form.savedValue = field.value;
      return;
    }
    if (field.name === "title" && field.value.trim() === "") {
      field.value = $("[data-base]", form).value;
      grow(field);
      return;
    }
    form.savedValue = field.value;
    form.pending = (form.pending || Promise.resolve()).then(() => post(form));
  }

  function adoptStamp(stamp) {
    if (!stamp || !app()) return;
    app().dataset.stamp = stamp;
    if (staleStamp === stamp) staleStamp = "";
  }

  function adoptBase(form, header) {
    if (header === null) return;
    const base = JSON.parse(header);
    $("[data-base]", form).value = base;
    if (form.matches(".details-form")) {
      const text = $("textarea[name=text]", form).value;
      for (const f of $$("form[action$='/task.description']:not([data-autosave])")) {
        $("input[name=base]", f).value = base;
        $("input[name=text]", f).value = text;
      }
    }
  }

  // keepalive lets a save outlive a navigation or the page itself, up to the 64 KB it allows.
  async function post(form) {
    const key = draftKey(form);
    const sent = form.savedValue;
    // Recorded before it leaves, so a refusal the page never hears of (it was closing) can't lose it.
    setDraft(key, { mine: sent, base: $("[data-base]", form).value });
    const body = new URLSearchParams(new FormData(form));
    let res;
    try {
      res = await fetch(form.action, {
        method: "POST",
        body,
        credentials: "same-origin",
        keepalive: body.toString().length < 60000,
        headers: { "HX-Request": "true", "X-Ordo-Page": app() ? app().dataset.stamp : "" },
      });
    } catch {
      form.savedValue = null;
      snack("Offline: the change didn't reach Ordo", "reload");
      return;
    }
    if (!res.ok) {
      form.savedValue = null;
      snack(res.status === 401 ? "Signed out" : "That didn't work. Check the connection.");
      return;
    }
    const trigger = res.headers.get("HX-Trigger");
    const sn = trigger && JSON.parse(trigger)["ordo:snack"];
    if (sn) snack(sn.msg, sn.undo);
    adoptStamp(res.headers.get("X-Ordo-Stamp"));
    // The page may have been swapped while the save was out.
    const live = liveForm(key);
    const theirs = res.headers.get("X-Ordo-Theirs");
    if (theirs !== null) {
      // A swapped-in field holds the file's text unless a draft was restored into it.
      const field = live && fieldOf(live);
      const mine = field && (live === form || field.value !== live.savedValue) ? field.value : sent;
      setDraft(key, { mine, theirs: JSON.parse(theirs) });
      if (live) showConflict(live);
    } else {
      setDraft(key);
      adoptBase(live || form, res.headers.get("X-Ordo-Base"));
    }
    if (res.body) res.body.cancel();
  }

  function flushSaves() {
    for (const form of $$("[data-autosave]")) save(form);
  }
  window.addEventListener("pagehide", flushSaves);

  document.addEventListener("change", (e) => {
    const t = e.target;
    const form = t.closest("[data-autosave]");
    if (form) save(form);
    if (t.matches("[data-attach-input]") && t.files.length) {
      snack("Uploading…");
      t.form.requestSubmit();
    }
  });

  document.addEventListener("focusout", () => setTimeout(refreshWhenIdle, 50));
  document.addEventListener("close", () => setTimeout(refreshWhenIdle, 50), true);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible") {
      if (source && source.readyState === EventSource.CLOSED) { source = null; listen(); }
    } else {
      flushSaves();
    }
  });

  document.addEventListener("submit", (e) => {
    const form = e.target;
    if (form.matches("[data-composer]")) {
      e.preventDefault();
      submitComposer(form.closest(".composer"), false);
    }
    if (form.matches("[data-separator-form]")) {
      e.preventDefault();
      finishSeparator(form.closest(".separator"), true);
    }
  }, true);

  document.body.addEventListener("ordo:snack", (e) => snack(e.detail.msg, e.detail.undo));
  document.body.addEventListener("ordo:focus", (e) => flash(e.detail.value));
  document.body.addEventListener("htmx:configRequest", (e) => {
    if (app()) e.detail.headers["X-Ordo-Page"] = app().dataset.stamp || "";
  });
  document.body.addEventListener("htmx:beforeRequest", (e) => {
    if (e.detail.requestConfig && e.detail.requestConfig.verb === "get") flushSaves();
  });
  document.body.addEventListener("htmx:afterRequest", (e) => {
    const xhr = e.detail.xhr;
    if (xhr && xhr.status < 300) adoptStamp(xhr.getResponseHeader("X-Ordo-Stamp"));
  });

  document.body.addEventListener("htmx:responseError", (e) => {
    snack(e.detail.xhr.status === 401 ? "Signed out" : "That didn't work. Check the connection.");
  });
  document.body.addEventListener("htmx:sendError", () => snack("Offline: the change didn't reach Ordo", "reload"));

  document.body.addEventListener("htmx:beforeSwap", (e) => {
    const cfg = e.detail.requestConfig;
    if (cfg && cfg.headers && cfg.headers["X-Ordo-Refresh"] && busy()) {
      e.detail.shouldSwap = false;
      staleStamp = "pending";
      return;
    }
    if (!e.detail.target || e.detail.target.id !== "app") return;
    stashDrafts();
    for (const c of $$(".composer")) closeComposer(c);
    for (const s of $$(".separator.editing")) {
      s.closed = true;
      if (s.watcher) s.watcher.destroy();
    }
    const sheet = $("dialog[open]");
    const left = Math.ceil(dismiss(sheet));
    if (left) e.detail.swapOverride = (e.detail.swapOverride || "outerHTML") + ` swap:${left}ms`;
  });

  // scannedToken returns the pairing token a scanned code carries, or "" for any other text.
  function scannedToken(text) {
    let s = text.trim();
    let url = null;
    try { url = new URL(s); } catch { }
    if (url) {
      const m = url.pathname.match(/\/pair\/([^/]+)\/?$/);
      if (!m) return "";
      s = m[1];
    }
    s = s.replace(/[\s-]/g, "").toUpperCase();
    return /^[A-Z2-7]{26,}$/.test(s) ? s : "";
  }

  let scanning = null;
  let jsQRLoad = null;

  function stopScan() {
    if (!scanning) return;
    scanning.stopped = true;
    clearTimeout(scanning.timer);
    if (scanning.stream) for (const track of scanning.stream.getTracks()) track.stop();
    const video = $("#scan-sheet video");
    if (video) video.srcObject = null;
    scanning = null;
  }

  function scanFailed(msg) {
    stopScan();
    const sheet = $("#scan-sheet");
    dismiss(sheet);
    const note = $("[data-scan-note]");
    if (note) {
      note.textContent = msg;
      note.hidden = false;
    }
    const field = $("[data-pair-token]");
    if (!field) return;
    if (sheet && sheet.open) sheet.addEventListener("close", () => field.focus(), { once: true });
    else field.focus();
  }

  function cameraError(err) {
    switch (err && err.name) {
      case "NotAllowedError":
      case "SecurityError":
        return "Camera access is blocked. Allow it in the browser's site settings, or paste the pairing link or code.";
      case "NotFoundError":
      case "OverconstrainedError":
        return "No camera found. Paste the pairing link or code instead.";
      case "NotReadableError":
        return "Another app is using the camera. Paste the pairing link or code instead.";
      default:
        return "The camera didn't start. Paste the pairing link or code instead.";
    }
  }

  // qrDecoder resolves to a function reading a QR code off a video frame, "" when there is none.
  async function qrDecoder(sheet) {
    if ("BarcodeDetector" in window) {
      try {
        if ((await BarcodeDetector.getSupportedFormats()).includes("qr_code")) {
          const detector = new BarcodeDetector({ formats: ["qr_code"] });
          return async (video) => {
            const codes = await detector.detect(video);
            return codes.length ? codes[0].rawValue : "";
          };
        }
      } catch { }
    }
    jsQRLoad = jsQRLoad || new Promise((resolve, reject) => {
      const script = document.createElement("script");
      script.src = sheet.dataset.decoder;
      script.onload = resolve;
      script.onerror = () => { jsQRLoad = null; reject(new Error("jsQR")); };
      document.head.append(script);
    });
    await jsQRLoad;
    const canvas = document.createElement("canvas");
    const ctx = canvas.getContext("2d", { willReadFrequently: true });
    return async (video) => {
      const scale = Math.min(1, 640 / Math.max(video.videoWidth, video.videoHeight));
      const w = Math.round(video.videoWidth * scale);
      const h = Math.round(video.videoHeight * scale);
      if (!w || !h) return "";
      canvas.width = w;
      canvas.height = h;
      ctx.drawImage(video, 0, 0, w, h);
      const code = jsQR(ctx.getImageData(0, 0, w, h).data, w, h, { inversionAttempts: "dontInvert" });
      return code ? code.data : "";
    };
  }

  async function startScan() {
    const sheet = $("#scan-sheet");
    const form = $("[data-pair-form]");
    if (!sheet || !form) return;
    if (!window.isSecureContext || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      scanFailed("The camera needs a secure (https) connection. Paste the pairing link or code instead.");
      return;
    }
    stopScan();
    const run = { stopped: false };
    scanning = run;
    const note = $("[data-scan-note]");
    if (note) note.hidden = true;
    const status = $("[data-scan-status]", sheet);
    status.textContent = "Starting the camera…";
    sheet.showModal();
    let stream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: { ideal: "environment" } } });
    } catch (err) {
      if (!run.stopped) scanFailed(cameraError(err));
      return;
    }
    if (run.stopped) {
      for (const track of stream.getTracks()) track.stop();
      return;
    }
    run.stream = stream;
    const video = $("video", sheet);
    video.srcObject = stream;
    video.play().catch(() => { });
    let decode;
    try {
      decode = await qrDecoder(sheet);
    } catch {
      if (!run.stopped) scanFailed("The scanner didn't load. Paste the pairing link or code instead.");
      return;
    }
    if (run.stopped) return;
    status.textContent = "Point the camera at the pairing code";
    const tick = async () => {
      let text = "";
      if (video.readyState >= 2) {
        try { text = await decode(video); } catch { }
      }
      if (run.stopped) return;
      const token = text && scannedToken(text);
      if (token) {
        stopScan();
        dismiss(sheet);
        $("[data-pair-token]", form).value = token;
        form.requestSubmit();
        return;
      }
      if (text) status.textContent = "That's not an Ordo pairing code";
      run.timer = setTimeout(tick, 120);
    };
    tick();
  }

  function copyText(text) {
    navigator.clipboard.writeText(text).then(() => snack("Link copied"), () => snack("Couldn't copy. Select the link instead."));
  }

  // signedInElsewhere leaves the login page once a device cookie arrived from another tab or the browser.
  function signedInElsewhere() {
    fetch("/projects", { redirect: "manual" })
      .then((res) => { if (res.ok) location.assign("/"); }, () => { });
  }

  document.addEventListener("click", (e) => {
    const t = e.target;
    if (t.closest("[data-scan]")) startScan();
    const copy = t.closest("[data-copy]");
    if (copy) copyText(copy.dataset.copy || location.href);
    const share = t.closest("[data-share]");
    if (share) navigator.share({ title: "Pair with Ordo", url: share.dataset.share }).catch(() => { });
  });
  document.addEventListener("close", (e) => { if (e.target.id === "scan-sheet") stopScan(); }, true);
  window.addEventListener("pagehide", stopScan);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") {
      const sheet = $("#scan-sheet");
      if (sheet && sheet.open) sheet.close();
    } else if (app() && app().matches("[data-login]")) {
      signedInElsewhere();
    }
  });
  document.addEventListener("submit", (e) => {
    const name = $("[data-device-name]");
    if (name && e.target.matches("[data-pair-form]")) e.target.elements.name.value = name.value;
  });

  function setupPairing(root) {
    for (const b of $$("[data-scan]", root)) b.hidden = false;
    for (const b of $$("[data-copy]", root)) b.hidden = !(navigator.clipboard && navigator.clipboard.writeText);
    for (const b of $$("[data-share]", root)) b.hidden = !navigator.share;
    const hint = $("[data-browser-hint]", root);
    if (hint) hint.hidden = matchMedia("(display-mode: standalone)").matches || navigator.standalone === true;
  }

  function init(root) {
    if (!root) return;
    updateBars();
    setupPairing(root);
    for (const t of $$("[data-autogrow]", root)) grow(t);
    adoptForms(root);
    setupSortable(root);
    syncSelection();
    listen();
  }

  document.body.addEventListener("htmx:afterSwap", (e) => {
    if (e.detail.target && e.detail.target.id === "app") adoptForms(app());
  });

  document.body.addEventListener("htmx:afterSettle", (e) => {
    if (!e.detail.target || e.detail.target.id !== "app") return;
    const root = app();
    init(root);
    const req = e.detail.requestConfig;
    const refreshed = req && req.headers && req.headers["X-Ordo-Refresh"];
    if (req && req.verb === "get" && !e.detail.pathInfo?.anchor && !refreshed) {
      if (location.hash) flash(location.hash.slice(1));
      else if (!staleStamp && req.triggeringEvent) window.scrollTo(0, 0);
    }
    if (afterRefresh) {
      const done = afterRefresh;
      afterRefresh = null;
      done();
    }
  });

  document.addEventListener("DOMContentLoaded", () => {
    init(app());
    if (location.hash) flash(location.hash.slice(1));
  });
})();
