---
id: tic-4kbm
status: closed
deps: []
links: []
created: 2026-05-26T00:33:02Z
type: task
priority: 2
assignee: gkarolyi
---
# center pane modals over dimmed preview

The command palette/help/dependency modals now render in the preview pane, which keeps the selected ticket visible.
They should be visually centered inside the preview pane instead of pinned near the top.

Keep the underlying preview visible behind the modal, but dim/fade it so it provides context without distracting from the modal.
Apply this consistently to command palette, help, create/query prompts, and dependency picker if they render in the preview pane.
