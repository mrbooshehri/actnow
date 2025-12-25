# actnow

Terminal-based task manager for the Eisenhower Matrix, focused on the Important & Immediate quadrant.

## Requirements

- Go 1.22+

## Build

```bash
go build -o actnow ./cmd/actnow
```

## Run

```bash
./actnow
```

Tasks are stored locally at `~/.actnow/tasks.json`.

## Keys

- `↑/↓` or `k/j`: Navigate
- `a`: Add task
- `e`: Edit task
- `d`: Mark done
- `x`: Delete
- `tab`: Next quadrant
- `q`: Quit

## Data model

Each task stores title, description, importance, urgency, due time, status, and creation time.
Urgency is automatically set if the due time is within 24 hours.
