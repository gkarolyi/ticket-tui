---
id: tic-20iw
status: closed
deps: []
links: []
created: 2026-05-25T23:14:35Z
type: task
priority: 1
assignee: gkarolyi
---
# UI fixes

the UI still has the following problems:
- layout is still horizontally split in a narrow terminal, so you can't see the list view properly
- command palette modal hides tha rest of the view underneath, so you can't see what ticket is selected
- likewise, the help menu hides the main UI
- the command palette list includes move selection commands, which are not needed here
- the command palette modal shrinks as you move down towards the bottom. it should be a fixed size and selection should scroll inside it
- the UI gets stale if tickets are updated elsewhere while the TUI is open. the list view should be refreshed periodically or when files change to accurately reflect the state of tickets

tips:
- look online at popular bubble tea apps for good UX examples. eg charmbracelet/glow, charmbracelet/huh, superfile
- use existing bubbles components where possible
- use lipgloss for styling
- make sure we're using the most up to date versions of libraries
- actually exercise the TUI and make sure it looks how you expect after making changes
