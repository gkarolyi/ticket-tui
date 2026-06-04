---
id: tic-nlw9
status: open
deps: [tic-bd5b, tic-4kbm]
links: []
created: 2026-05-26T00:33:02Z
type: task
priority: 2
assignee: gkarolyi
---
# support mouse clicks in TUI

Add mouse support for the TUI.

Expected interactions:
- click a ticket row to select it
- click command/help/dependency modal rows or controls where practical
- preserve keyboard-first behavior and existing keybindings

Use Bubble Tea v2's declarative mouse support via `tea.View` fields and v2 mouse message types.
This should build on the finalized layout and modal positioning work so click regions match what is rendered.
