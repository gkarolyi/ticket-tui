# ticket-tui

A Bubble Tea terminal UI plugin for [`ticket`](https://github.com/wedow/ticket).

`ticket-tui` is a standalone Go project that builds the `ticket-tui` binary. When that binary is in your `PATH`, `ticket` can launch it as a plugin with:

```bash
tk tui
```

## What it does

`ticket-tui` gives `ticket` installs an interactive dashboard for:

- browsing ready, active, blocked, and recently closed tickets
- opening a large centered reader modal for ticket content
- creating tickets from the TUI
- editing tickets with `tk edit`
- starting, closing, and reopening tickets
- adding dependencies from the TUI
- filtering tickets by id, title, or description

The UI reads ticket files directly and shells out to the existing `ticket` CLI for mutations, so the source of truth stays in your installed `ticket` workflow.

## Requirements

You need an existing `ticket` installation.

`ticket-tui` expects to be launched through `tk tui`, which provides:

- `TICKETS_DIR` — path to the `.tickets` directory
- `TK_SCRIPT` — absolute path to the `ticket` / `tk` script

Direct invocation is also possible if those environment variables are set manually.

## Install

Build from source:

```bash
git clone https://github.com/gkarolyi/ticket-tui.git
cd ticket-tui
go build ./cmd/ticket-tui
```

Then place the resulting binary somewhere in your `PATH`.

Example:

```bash
install -m 0755 ./ticket-tui ~/.local/bin/ticket-tui
```

If `ticket` is already installed and `ticket-tui` is in your `PATH`, this will work:

```bash
tk tui
```

## Usage

```bash
ticket-tui --help
ticket-tui --tk-describe
```

In normal use you launch it through `ticket`:

```bash
tk tui
```

## Key bindings

Main screen:

- `j` / `k` — move selection
- `tab` — jump to next section
- `enter` — open centered reader modal
- `n` — create ticket
- `d` — add dependencies
- `e` — edit selected ticket
- `s`, `c`, `r` — start, close, reopen
- `/` — filter tickets
- `ctrl+p` — command palette
- `R` — refresh
- `?` — help
- `q` — quit

Reader modal:

- `j` / `k` — scroll
- `e` — edit selected ticket
- `esc` — close reader and return to dashboard

## Development

Run tests:

```bash
go test ./...
```

## License

MIT
