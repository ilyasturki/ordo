# The app's look

The Android app's settled look decisions, one line each. Later work starts from them; changing one is a decision of its own.

- **Colour**: Material 3 dynamic colour, light and dark following the system, no toggle; red and teal (`marks.up`, `marks.down`) only for priority, the tag colours only for tags.
- **Type**: Google Sans Flex for every role.
- **Motion**: the platform's defaults (`MotionScheme.standard()`); only what follows the finger (the pager, a drag) and list changes (`animateItem`) move otherwise.
- **Chips**: one outlined chip for a task's values (`MarkChip`): 1dp `outline` border, no fill, the priority tint kept; 24dp and read only in the list, 32dp where a tap opens its picker (add sheet, task page); the add bar keeps a set value compact (the arrow, the dot, the estimate) to stay on one line. An unset value is its bare icon, except the task page's "Colour".
- **Separators**: 3dp terminal-style bars in `onSurface`, with the label in bold.
- **Text edge**: 16dp for every screen's text and chips; a row with a status or leading icon puts its text at 52dp, in the list and on the task page alike.
- **Top bar**: the focused category's name as Home's title, the project tabs on their own line; no divider, the bar takes `surfaceContainer` once the page in sight scrolls under it.
- **Hidden gestures**: each has a visible way too; the project's actions sit behind Home's ⋮ (with Settings), and a tap on the open tab or a long press stays a shortcut.
- **Snackbar**: on the bottom edge, through `OrdoApp`'s single collector, 4 seconds; the `+` (or "Mark done") rises above it while it shows.
- **Add sheet**: one row under the title, at 360dp whatever is set: the category, icons for the unset fields, chips for the set ones, then Save; it stays open after Save.
- **Sheet actions**: 56dp list items (`SheetAction`), icon and label, red for a destructive one.
- **Empty screens**: one centred line in `onSurfaceVariant` (`EmptyState`), with a tonal button when there is a way out the screen doesn't already offer.
