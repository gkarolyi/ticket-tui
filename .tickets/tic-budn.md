---
id: tic-budn
status: closed
deps: []
links: []
created: 2026-05-25T06:45:16Z
type: task
priority: 2
assignee: gkarolyi
---
# width-aware layout

right now the list pane and the view pane are always side by side.
in a narrow window, this makes both the list and the view less readable.
it would be better if the layout was split horizontally when the terminal is wide,
and split vertically when the terminal is narrow. 
we should make the tui aware of the terminal size. 
the status bar at the bottom is also too wide right now and i can't see all the text because it
disappears off screen to the right. we will make changes to the status bar elsewhere, but this
ticket should make the app properly terminal size aware. i believe bubble tea has components/
helpers to deal with this.
