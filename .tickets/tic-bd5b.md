---
id: tic-bd5b
status: closed
deps: []
links: []
created: 2026-05-26T00:33:02Z
type: task
priority: 2
assignee: gkarolyi
---
# stack TUI layout below 150 columns

The current TUI layout switches from side-by-side panes to stacked panes around 100 columns.
That still feels cramped in typical medium-width terminals.

Change the responsive breakpoint so the list/detail layout stacks below 150 columns.
Verify the list remains readable around 120-140 columns and the side-by-side layout still works in wider terminals.
