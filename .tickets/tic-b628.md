---
id: tic-b628
status: open
deps: []
links: []
created: 2026-05-25T07:29:40Z
type: task
priority: 0
assignee: gkarolyi
---
# git hook

we need a precommit git hook that runs all formatting and tests before every commit.
use all standard go tools, like vet and lsp etc, without installing any additional ones.
each commit should be a well formatted, idiomatic piece of work that passes tests.
