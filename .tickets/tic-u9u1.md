---
id: tic-u9u1
status: closed
deps: []
links: []
created: 2026-05-25T07:04:17Z
type: task
priority: 2
assignee: gkarolyi
---
# split off ticket-tui from main ticket repo

i won't be submitted a PR against the main ticket repo because it's self contained and written
in bash.
this repo should just be a simple go repo for the ticket-tui binary, which people can then copy
to their PATH to use as a plugin.
