---
id: tic-ytfw
status: closed
deps: []
links: []
created: 2026-05-26T00:33:01Z
type: task
priority: 1
assignee: gkarolyi
---
# UX overhaul

the current TUI is very bare bones and doesn't look professional or delightful to use.
it's fine for now, but i think we can improve on it.
fetch the tui-design skill and references from https://github.com/hyperb1iss/hyperskills/tree/main/skills/tui-design, we will be using these to design a new overhauled interface.
interview me based on the skill and use this ticket to document decisions about the design.
show me different examples visually so we can agree on the right direction.

## Design discovery

- Primary workflow: project cockpit. The TUI should make broader project context visible, including dependencies, relationships, notes, and status distribution.
- Visual direction: lazygit-dense. Favor fixed spatial panels, compact keyboard affordances, and high information value over decorative chrome.
- Default density: balanced. Show enough metadata for decisions without crowding the screen.
- First implementation slice: both lightly. Make a cohesive first pass across layout, visual hierarchy, focus states, footer/status, and modal behavior before deeper interaction work.

## TUI design references

- Layout baseline: persistent multi-panel dashboard, inspired by lazygit/lazydocker.
- Interaction baseline: keyboard-first, contextual footer, command palette, `?` help overlay, `/` search/query, `Esc` back/cancel.
- Visual baseline: semantic color slots, active focus indicator, muted inactive panels, content-first borders.
- Responsive baseline: minimum usable size, stacked layout below an agreed breakpoint, no crashes on resize.
- Accessibility baseline: usable without color, respect `NO_COLOR`, do not rely on Nerd Fonts.

## Candidate visual directions

### Option A: Project Cockpit

Persistent left navigation and a rich selected-ticket workspace.

```text
┌─ tk · ticket cockpit ─────────────── mode: Active ───── 12 open · 3 blocked · 1 in progress ┐
│ ┌─ Tickets ───────────────┐ ┌─ tic-ytfw · UX overhaul ─────────────────────────────────────┐ │
│ │ > P1 open   UX overhaul │ │ Status  open        Priority  P1        Assignee gkarolyi    │ │
│ │   P1 open   stack TUI…  │ │ Created 2026-05-26  Links     -         Deps     clear       │ │
│ │   P1 open   center…     │ ├─ Relationships ─────────────────────────────────────────────┤ │
│ │   P2 open   split TUI…  │ │ blocked by: -       blocks: tic-nlw9     linked: -           │ │
│ │   P2 open   changelog   │ ├─ Notes ─────────────────────────────────────────────────────┤ │
│ │                          │ │ # UX overhaul                                                │ │
│ └──────────────────────────┘ │ current TUI is bare bones...                                  │ │
│ ┌─ Modes ─────────────────┐ │                                                              │ │
│ │ Active  All  Closed     │ │                                                              │ │
│ └──────────────────────────┘ └──────────────────────────────────────────────────────────────┘ │
└─ 4 tickets | [j/k]move [tab]mode [enter]inspect [n]new [/]query [ctrl+p]palette [?]help ─────┘
```

### Option B: Lazygit-Style Board

More compact left-side context panels, with the detail pane reserved for ticket content.

```text
┌─ tk ─────────────────────────────────────────────────────────────────────────────────────────┐
│ ┌─ Ready ────────────────┐ ┌─ Detail: tic-ytfw ────────────────────────────────────────────┐ │
│ │ > P1 tic-ytfw UX…      │ │ UX overhaul                                                   │ │
│ │   P1 tic-bd5b stack…   │ │                                                              │ │
│ │   P1 tic-4kbm modal…   │ │ Metadata                                                     │ │
│ ├─ In Progress ──────────┤ │ Status: open     Priority: P1     Assignee: gkarolyi          │ │
│ │   none                 │ │                                                              │ │
│ ├─ Blocked ──────────────┤ │ Relationships                                                │ │
│ │   tic-nlw9 mouse       │ │ blocks -> tic-nlw9 [open] support mouse clicks in TUI        │ │
│ ├─ Closed Recent ────────┤ │                                                              │ │
│ │   tic-b628 git hook    │ │ the current TUI is very bare bones...                        │ │
│ └────────────────────────┘ └──────────────────────────────────────────────────────────────┘ │
└─ Active panel: Ready | [tab]focus [s]start [c]close [d]deps [e]edit [R]refresh [?]help ──────┘
```

### Option C: Focused Cockpit

Single primary list with a high-signal right pane and a compact command/status rail.

```text
┌─ tk tickets ──────────────────────────────────────────────────────────────── Project cockpit ┐
│ P1  open          tic-ytfw  UX overhaul                                                       │
│ P1  open          tic-bd5b  stack TUI layout below 150 columns                                │
│ P1  open          tic-4kbm  center pane modals over dimmed preview                            │
│ P2  open          tic-u9u1  split off ticket-tui from main ticket repo                        │
│                                                                                               │
├─ Selected ───────────────────────────────┬─ Actions ─────────────────────────────────────────┤
│ UX overhaul                              │ n new       s start      c close      r reopen     │
│ Status: open · Priority: P1              │ d deps      e edit       / query      ctrl+p cmds  │
│ Blocks: tic-nlw9                         │ R refresh   tab mode     ? help       q quit       │
│ Notes preview starts here...             │ Last: 4 tickets · refreshed 5s ago                │
└──────────────────────────────────────────┴──────────────────────────────────────────────────┘
```

## Open design questions

- Which option should be the base direction?
- Should status distribution appear in the header, a side panel, or only in the footer/status line?
- Should ticket modes remain `tab` cycling, or become explicit focusable panels?

## Design decisions

- Base direction: Option B, lazygit-style board.
- Status distribution: side panel by default, so project health is visible without consuming header width.
- Mode UX: keep current `tab` mode cycling for now; do not introduce focusable panels in the first slice.

## First implementation slice

- Replace the plain list/detail split with a board-style left column containing ticket rows plus project status sections.
- Keep the right pane as the selected ticket detail surface.
- Add clearer active panel/title treatment and contextual footer copy.
- Preserve existing keyboard behavior and command palette actions.

## First slice visual validation

- Validation tool: run `ticket-tui` in `tmux` with a fixed terminal size and capture the pane output.
- First capture at 120x30 exposed a real layout issue: long rows in the left board ran visually into the detail pane, producing joined text such as `tickId:` and `improvementType:`.
- Fix applied: side-by-side layouts now reserve a two-column gutter between the board and detail pane.
- Second capture at 120x30 showed the panes separated cleanly, but the broader interaction and information-design issues below remain.

## User feedback after first slice

- Current mode is not discoverable enough. The user cannot tell which modes exist, what the current mode means, or what the next mode will be.
- The `Ready` / `In Progress` / `Closed Recent` section navigation is confusing. Arrow movement skips empty/non-selected sections in surprising ways, and `up` from the top can jump to another section. These should behave like one clear category-aware list that navigates normally with arrow keys.
- `tab` should jump to the first ticket in the next visible section, not cycle global modes as the primary section navigation mechanism.
- The preview pane cannot be scrolled, so long tickets cannot be read without editing them.
- Ticket rows spend too much width on id, priority, and status. Those fields should be more compact so more title text is visible.
- Pressing `enter` on a ticket should open a full-screen reading view for that ticket.
- Long lines do not wrap in the preview pane, so detail content is lost horizontally.
- The metadata section is messy. Attributes should use a fixed table-style layout so they are easy to scan.
- Help and command palette overlays should be more centered and should not completely hide the content behind them. Look at Crush or other polished Bubble Tea apps for better overlay patterns.
- Markdown preview is bare bones. Evaluate `github.com/charmbracelet/glamour` for styled markdown rendering.

## Refinement guidance from tui-design skill

- Treat this as a refinement loop: validate against anti-patterns, adjust layout/interaction/visual system, then re-run visual captures.
- Prioritize discoverability: contextual footer, visible mode/category state, and `?` help should make available navigation clear.
- Preserve spatial consistency: sections should stay predictable, and movement should not surprise the user.
- Use progressive disclosure: compact row data in the board, richer metadata and markdown in the detail/full-screen reader.
- Validate at multiple terminal sizes: 80x24, 120x30 or 120x40, and a wide layout such as 200x60.
- Validate with color removed or reduced; hierarchy must remain understandable without relying on color alone.

## Proposed refinement slices

1. Navigation model: replace global mode cycling as the main mental model with a section-aware list. Arrow keys move through visible tickets normally; `tab` jumps to the first ticket in the next visible section; header/footer show available sections and current position.
2. Reading model: make the preview pane scrollable with existing viewport controls, wrap long lines to the pane width, and add `enter` full-screen reader mode with `Esc` back.
3. Row density and metadata: compact ticket row badges for priority/status/id, maximize title width, and render metadata as fixed-width aligned fields.
4. Overlay polish: center help and command palette over a dimmed/visible background instead of replacing the detail pane content.
5. Markdown rendering: evaluate and, if appropriate, integrate Charmbracelet Glamour for styled ticket body preview with terminal-width-aware rendering.

## Refinement slice 1: section navigation

- Changed `tab` from global mode cycling to next-section navigation.
- Arrow-key movement now follows the rendered section order, so moving down from the last ready ticket goes to the first in-progress ticket before closed tickets.
- Footer now shows the current section and `tab next section` instead of the opaque `tab mode` hint.
- Visual validation: tmux 120x30 capture after pressing `tab` showed selection in `In Progress` and footer text `section In Progress`.
- Remaining issue discovered during validation: the header still says `mode: active`, which is less helpful now that section navigation is the main model. The next refinement should replace or clarify that header state.

## Refinement slice 2 clarification: preview focus and reading

- Keep the header follow-up, but prioritize preview readability next.
- Long preview lines must wrap so the end of each line remains visible inside the preview pane.
- Long ticket content must be scrollable without opening the external editor.
- Pressing `enter` on a ticket should switch focus to the preview pane for reading, using the styled Glamour viewer.
- When the preview pane is focused, `up`/`down` and `k`/`j` scroll the preview instead of moving through the ticket list.
- In preview focus, `e` should still open the selected ticket in the external editor.
- In preview focus, `esc` should switch focus back to the ticket list.

## Refinement slice 2: preview focus, wrapping, and scrolling

- Added preview focus state. Pressing `enter` on a selected ticket focuses the preview pane; `esc` returns focus to the ticket list.
- When preview is focused, `j`/`k` and arrow keys scroll the preview instead of changing ticket selection.
- The footer now switches to `preview focused | j/k scroll  e edit  esc ticket list` while reading.
- Detail rendering now receives the preview pane width so long body lines wrap instead of disappearing past the right edge.
- Added Glamour markdown rendering for width-aware styled preview output. `github.com/charmbracelet/glamour` v0.8.1 was used because v1.0.0 conflicted with the current Charm dependency graph; `github.com/charmbracelet/x/cellbuf` was updated to v0.0.15 to keep dependencies compiling.
- Fixed a detail/selection mismatch discovered during visual validation: after section ordering, the initial preview could show a different ticket than the selected row.
- Visual validation: tmux 120x30 capture after `enter` and preview scroll showed the selected `tic-u9u1` row matched `Detail: tic-u9u1`, long content wrapped in the preview, and the preview-focused footer was visible.

## Regression fix: startup query prompt from Glamour auto style

- Bug: opening the TUI could immediately show the `Query Tickets` prompt with random terminal-response garbage in the text input.
- Root cause: Glamour was configured with `WithAutoStyle()`. Auto style can query the terminal for style/color information; terminal responses arrive on stdin and Bubble Tea can interpret them as user input, including `/`, which opens query mode.
- Regression coverage: added a test that Glamour preview rendering uses a fixed style instead of `auto`, plus a startup view test that loaded state does not open the query prompt.
- Fix: switched Glamour rendering to `WithStandardStyle("dark")`, avoiding terminal probing during preview rendering.
- Dependency update: upgraded to the latest available `github.com/charmbracelet/glamour` version, `v1.0.0`. Keeping `github.com/charmbracelet/x/cellbuf` at `v0.0.15` is required for this dependency graph to compile.
- Visual validation: tmux 120x30 startup capture showed the normal ticket board/detail view, not the query prompt.

## Refinement slice 3: preview focus affordance and responsive footer

- Preview focus needs a visible affordance so the user can tell when arrow keys scroll the preview instead of moving ticket selection.
- The affordance should stay minimal: a brighter/active detail title is enough for now; avoid heavy borders until overlay/layout polish.
- Footer text must adapt to narrow widths by showing the most important actions first instead of simply truncating the right side.
- Narrow preview focus footer priority: show `preview`, `esc list`, and `j/k scroll` before lower-priority actions such as `e edit`.
- Implemented: preview focus changes the detail title from `Detail: <id>` to `Preview: <id>` and applies active styling to the title.
- Implemented: preview-focused footer now prioritizes `preview | esc list  j/k scroll  e edit`, so the escape hint remains visible on narrow screens.
- Visual validation: tmux captures at 120x30 and 70x24 showed `Preview: tic-ytfw` and the responsive preview footer.

## Follow-up fix: narrow main footer and refresh detail sync

- User feedback: the main ticket-list footer was still too long on narrow displays.
- User feedback: after leaving preview focus, the detail pane could show content for a different ticket than the selected row.
- Root cause: compact footer text still went through the wide status-prefix path, duplicating status and truncating the tail on narrow screens.
- Root cause: refresh/load rebuilt preview content from the first ordered ticket while preserving the selected index, so the selected row and preview body could diverge after refresh.
- Implemented: narrow main footer now uses compact hints such as `16 tickets | Ready | q quit | ? help | j/k move | tab section`.
- Implemented: refresh now loads detail for the clamped selected ticket in the ordered ticket list.
- Regression coverage: added tests for narrow main footer content and selected-ticket detail preservation across refresh.
- Visual validation: tmux captures at 70x24 and 100x28 confirmed the footer fits and selected/detail IDs remain aligned after preview refresh and escape.
