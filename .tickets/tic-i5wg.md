---
id: tic-i5wg
status: closed
deps: []
links: []
created: 2026-05-28T02:26:29Z
type: feature
priority: 1
assignee: gkarolyi
---
# ticket-tui dashboard UX redesign

Design a clearer dashboard-first UX for ticket-tui, using hyperskills tui-design guidance and gh-dash as a quality reference.

## Design

Adopt a stable dashboard layout with an overview header, sectioned actionable list, concise metadata-first detail pane, clearer browse vs inspect states, and responsive narrow-screen behavior.

## Acceptance Criteria

A written spec captures the chosen direction, mockups for standard and narrow terminals exist, and ticket notes record the design decisions and rationale.


## Notes

**2026-05-28T02:28:03Z**

Design decisions: use a dashboard-first two-pane layout based on Option A. Keep Ready/In Progress/Blocked/Closed Recent as visible queue sections, add a compact counts header, make ticket rows title-first with compact priority/status tokens, and make the detail pane metadata-first before long-form markdown. Browse and inspect remain separate states; enter focuses a readable preview and esc returns to the queue. Responsive targets: wide 120+ columns, standard 100-119 columns, minimum supported 80-99 columns with a stacked layout below the header. Spec written at docs/superpowers/specs/2026-05-28-ticket-tui-dashboard-ux-design.md.

**2026-05-28T02:36:12Z**

Spec refinement: metadata should use column headers in a compact horizontal grid for standard and wide layouts, with responsive wrapping into compact groups at minimum supported widths. Added an explicit verification approach: fixture-backed view tests at 120x40, 100x28, and 80x24; golden mockup comparisons for key states; manual terminal captures for browse/inspect/overlay states; and resize plus NO_COLOR validation.

**2026-06-04T00:02:12Z**

Implementation refinements completed after spec approval: aligned queue columns across sections; filtered preview body to show Notes when present and otherwise first freeform content while excluding duplicated relationship sections; added blank-line separation between pinned summary and body; kept selected list rows visible while scrolling; changed enter/escape flow to use a centered reader modal over the dashboard; fixed overlay regressions so help, palette, and preview overlays preserve underlying formatting and do not corrupt visible preview text. Verified repeatedly with go test ./internal/tui, go test ./..., and fixed-size tmux captures of help and reader modal states.
