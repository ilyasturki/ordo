// The README's phone screens, in the order run.sh lays them side by side (format in testdata/web/shots.mjs).
const P = "f3a1c0de-2b7e-4c19-9d51-8e6f0a4b7c21";
// The past versions, folded as assets/demo/state.json folds them in the TUI demo.
const prefs = { current: P, folded: Object.fromEntries(["c-future", "c-v100", "c-v101", "c-v110", "c-v120", "c-v121"].map((c) => [c, true])) };

export default [
  { name: "list", path: "/", prefs },
  { name: "task", path: `/p/${P}/t/t-a1`, ready: "textarea[aria-label=Title]" },
  { name: "add", path: "/", prefs, steps: [
    { at: "[data-cat] button[aria-label^='Add a task to']", release: true },
    { text: "Retry uploads on a flaky connection" },
    "(() => { if (!document.activeElement.matches('[data-cat] input[aria-label=\"New task\"]') || document.activeElement.value !== 'Retry uploads on a flaky connection') throw new Error('the add row does not hold the typed title'); })()",
  ] },
];
