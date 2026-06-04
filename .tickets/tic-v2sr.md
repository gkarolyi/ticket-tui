---
id: tic-v2sr
status: closed
deps: []
links: []
created: 2026-05-29T00:41:05Z
type: task
priority: 2
assignee: gkarolyi
---
# add description to create ticket modal

it would be nice to be able to quickly add a description and maybe priority/dependencies to a new ticket
when first creating it in the modal in the TUI.
check out patterns in gh-dash and the tui-design skill (https://github.com/hyperb1iss/hyperskills/tree/main/skills/tui-design) for inspiration

## Notes

**2026-06-04T00:16:22Z**

Implemented a two-step create modal in ticket-tui. Pressing n now opens a title step first, then advances to a description step on enter. Submitting the second step calls tk super create <title> -d <description> when description is non-empty. Esc cancels from either step. Verified with go test ./internal/tui and go test ./....
