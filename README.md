# actnow

ActNow is a fast, keyboard-driven terminal app for managing tasks with the Eisenhower Matrix, focused on the Important & Immediate quadrant.

## Features

- 2x2 quadrant view with per-quadrant task lists
- Fast add/edit with a centered modal
- Important/Urgent classification with quadrant-specific fields
- Due/SLA and planned date pickers
- Local JSON persistence (offline-first)
- Keyboard-only workflow

## Install

```bash
go build -o actnow ./cmd/actnow
```

## Run

```bash
./actnow
```

Data is stored at `~/.actnow/tasks.json`.

## Keys (Main)

- `↑/↓` or `j/k`: Move between tasks
- `tab`: Next quadrant
- `shift+tab`: Previous quadrant
- `a`: Add task
- `e`: Edit task
- `d`: Toggle done/undone
- `x`: Delete task
- `h`: Help
- `q`: Quit

## Keys (Add/Edit)

- `↑/↓` or `j/k`: Move between fields
- `i`: Insert mode for text fields
- `enter`: Next field (saves on last)
- `space`: Toggle checkboxes
- `esc`: Exit insert mode or close form
- Date fields: `h/l` move segment, `+/-` change value, `t` current time, `x` clear

## Quadrant Fields

- Important & Immediate: status, title, due/SLA, impact, next action
- Important & Not Immediate: status, title, planned date, effort estimate
- Not Important & Immediate: status, title, due/SLA, delegate to
- Not Important & Not Immediate: title, delete reason

## Examples

- I+I: Title: Fix prod outage, Impact: Revenue loss, Next Action: Restart DB, Due/SLA: 2025-01-05 13:00
- I+NI: Title: Write migration plan, Planned Date: 2025-01-12 09:00, Effort: 4h
- NI+I: Title: Renew SSL cert, Delegate To: ops@team, Due/SLA: 2025-01-07 10:00
- NI+NI: Title: Remove old test data, Delete Reason: Not needed

## License

MIT. See `LICENSE`.
