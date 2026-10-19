"use strict";

(() => {
  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));
  const app = () => $("#app");

  const selected = new Set();
  let selecting = false;
  let dragging = false;
  let staleStamp = "";
  let keepAdding = null;
  let snackTimer = 0;

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
    return dragging || $("dialog[open]") || (a && a.matches("input:not([type=hidden]), textarea, select"));
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

  function openAdd(cat) {
    const sheet = $("#add-sheet");
    if (!sheet) return;
    const radio = $(`input[name=cat][value="${CSS.escape(cat)}"]`, sheet) || $("input[name=cat]", sheet);
    if (radio) {
      radio.checked = true;
      radio.closest("label").scrollIntoView({ inline: "center", block: "nearest" });
    }
    sheet.showModal();
    const input = $("input[name=title]", sheet);
    input.value = "";
    input.focus();
  }

  function grow(el) {
    el.style.height = "auto";
    el.style.height = el.scrollHeight + "px";
  }

  let sortables = [];
  function setupSortable(root) {
    sortables = [];
    if (!window.Sortable || root.dataset.page !== "project") return;
    for (const list of $$(".content .rows", root)) {
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
  document.addEventListener("contextmenu", (e) => { if (e.target.closest(".content .row")) e.preventDefault(); });

  document.addEventListener("click", (e) => {
    const t = e.target;
    if (pressOrigin && pressOrigin.fired) {
      pressOrigin = null;
      if (t.closest(".content .row")) {
        e.preventDefault();
        e.stopPropagation();
        return;
      }
    }
    pressOrigin = null;
    const fold = t.closest("[data-fold]");
    if (fold) {
      const section = fold.closest(".category");
      const folded = section.classList.toggle("folded");
      fold.setAttribute("aria-expanded", folded ? "false" : "true");
      fetch("/fold/" + encodeURIComponent(fold.dataset.fold), { method: "POST", credentials: "same-origin" });
      return;
    }
    const add = t.closest("[data-add-to]");
    if (add) {
      openAdd(add.dataset.addTo);
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
    const open = t.closest("[data-sheet-open]");
    if (open) {
      const d = document.getElementById(open.dataset.sheetOpen);
      if (d) d.showModal();
      return;
    }
    if (t.closest("[data-sheet-close]")) {
      keepAdding = null;
      t.closest("dialog").close();
      return;
    }
    if (t instanceof HTMLDialogElement && t.open) {
      const r = t.getBoundingClientRect();
      if (e.clientY < r.top || e.clientY > r.bottom || e.clientX < r.left || e.clientX > r.right) {
        keepAdding = null;
        t.close();
      }
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
  });

  document.addEventListener("input", (e) => {
    const t = e.target;
    if (t.matches("[data-autogrow]")) grow(t);
    const form = t.closest("[data-autosave]");
    if (form && t.matches("textarea[name=text]")) {
      clearTimeout(form.saveTimer);
      form.saveTimer = setTimeout(() => save(form), 1200);
    }
  });

  function save(form) {
    clearTimeout(form.saveTimer);
    const field = $("textarea[name=title], textarea[name=text]", form);
    if (!field || field.value === form.savedValue) return;
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
    adoptBase(form, res.headers.get("X-Ordo-Base"));
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
    if (form.matches("[data-add-form]")) {
      const cat = $("input[name=cat]:checked", form);
      keepAdding = cat ? cat.value : null;
    }
  });

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
    const sheet = $("dialog[open]");
    if (sheet && e.detail.target && e.detail.target.id === "app") sheet.close();
  });

  // scannedToken returns the pairing token a scanned code carries, or "" for any other text.
  function scannedToken(text) {
    let s = text.trim();
    let url = null;
    try { url = new URL(s); } catch { }
    if (url) {
      const m = url.pathname.match(/^\/pair\/([^/]+)$/);
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
    if (sheet && sheet.open) sheet.close();
    const note = $("[data-scan-note]");
    if (note) {
      note.textContent = msg;
      note.hidden = false;
    }
    const field = $("[data-pair-token]");
    if (field) field.focus();
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
      stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: { ideal: "environment" } }, audio: false });
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
        sheet.close();
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
    fetch("/projects", { credentials: "same-origin", redirect: "manual" })
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
    for (const f of $$("[data-autosave]", root)) {
      const field = $("textarea", f);
      f.savedValue = field ? field.value : "";
    }
    setupSortable(root);
    syncSelection();
    listen();
  }

  document.body.addEventListener("htmx:afterSettle", (e) => {
    if (!e.detail.target || e.detail.target.id !== "app") return;
    const root = app();
    init(root);
    const req = e.detail.requestConfig;
    if (req && req.verb === "get" && !e.detail.pathInfo?.anchor) {
      if (location.hash) flash(location.hash.slice(1));
      else if (!staleStamp && req.triggeringEvent) window.scrollTo(0, 0);
    }
    if (keepAdding && root.dataset.page === "project") {
      const cat = keepAdding;
      keepAdding = null;
      openAdd(cat);
      keepAdding = null;
    }
  });

  document.addEventListener("DOMContentLoaded", () => {
    init(app());
    if (location.hash) flash(location.hash.slice(1));
  });
})();
