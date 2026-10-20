// The README's phone screens, in the order run.sh lays them side by side (format in testdata/web/shots.mjs).
const P = "/p/f3a1c0de-2b7e-4c19-9d51-8e6f0a4b7c21";
// The past versions, folded as assets/demo/state.json folds them in the TUI demo.
const fold = "c-future.c-v100.c-v101.c-v110.c-v120.c-v121";

export default [
  { name: "list", path: P, fold },
  { name: "task", path: `${P}/t/t-a1` },
  { name: "add", path: P, fold, steps: [
    "document.querySelector('[data-add-fab]').click()",
    "document.querySelector('.composer-input').value = 'Retry uploads on a flaky connection'",
  ] },
];
