---
id: tic-9wqr
status: closed
deps: []
links: []
created: 2026-05-25T06:49:13Z
type: task
priority: 2
assignee: gkarolyi
---
# command palette

we should add a command palette. it needs to have the following properties:
- pops up in a modal
- modal should be positioned so that selected item is still visible
- lists all available commands (ie all functions exposed by tk and this tui)
- commands are fuzzy searchable
- on enter, the selected function is executed
