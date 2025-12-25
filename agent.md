# Agent Specification: Important–Immediate Quadrant (Terminal App)

## Overview

You are an autonomous software agent responsible for **designing, implementing, testing, and documenting** a **terminal-based application written in Go (Golang)** that helps users manage tasks using the **Eisenhower Matrix**, with a strong focus on the **Important & Immediate (Urgent)** quadrant.

The application must be fast, keyboard-driven, offline-first, and suitable for power users (DevOps, engineers, sysadmins).

---

## Goal

Build a **TUI (Terminal User Interface)** application that allows users to:

* Capture tasks quickly
* Automatically classify tasks into Eisenhower quadrants
* Prioritize and manage **Important & Immediate** tasks efficiently
* Review, complete, defer, or delegate tasks
* Persist data locally (no cloud dependency)

Primary focus:

> **Important + Immediate = Act Now**

---

## Core Concepts

### Eisenhower Matrix

Tasks are categorized by:

* **Importance**: contributes to long-term goals, consequences
* **Urgency**: requires immediate attention

Quadrants:

1. **Important & Immediate** (🔥 Focus of this app)
2. Important & Not Immediate
3. Not Important & Immediate
4. Not Important & Not Immediate

---

## Scope (MVP)

### Must Have

* Terminal UI (TUI)
* CRUD for tasks
* Task metadata:

  * Title
  * Description
  * Importance (bool)
  * Urgency (bool)
  * Due time (optional)
  * Created at
  * Status (pending / done / deferred)
* Persistent storage (local file)
* Dedicated view for **Important & Immediate** tasks
* Keyboard shortcuts

### Nice to Have (Optional)

* Notifications (terminal bell)
* Task aging (auto-mark urgent)
* Export to Markdown
* Daily review mode

---

## Technical Constraints

* Language: **Go ≥ 1.22**
* OS: Linux / macOS (Windows optional)
* No external services
* Minimal dependencies

---

## Architecture

### High-Level Design

```
+--------------------+
| Terminal UI (TUI)  |
|  BubbleTea / Tview|
+---------+----------+
          |
+---------v----------+
| Application Logic  |
|  Task Engine       |
|  Quadrant Rules    |
+---------+----------+
          |
+---------v----------+
| Persistence Layer  |
|  JSON / BoltDB     |
+--------------------+
```

---

## Technology Choices

### UI

Choose **one**:

* `github.com/charmbracelet/bubbletea` (preferred)
* `github.com/rivo/tview`

### Storage

Choose **one**:

* JSON file in `$HOME/.actnow/tasks.json`
* BoltDB / bbolt

---

## Data Model

### Task Struct

```go
type Task struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Important   bool      `json:"important"`
    Urgent      bool      `json:"urgent"`
    DueAt       *time.Time `json:"due_at,omitempty"`
    Status      string    `json:"status"` // pending, done, deferred
    CreatedAt   time.Time `json:"created_at"`
}
```

---

## Business Rules

### Quadrant Detection

```go
func Quadrant(t Task) string {
    switch {
    case t.Important && t.Urgent:
        return "Important & Immediate"
    case t.Important:
        return "Important & Not Immediate"
    case t.Urgent:
        return "Not Important & Immediate"
    default:
        return "Not Important & Not Immediate"
    }
}
```

### Urgency Escalation

* If `DueAt` is within **24 hours**, mark `Urgent = true`

---

## Terminal UX

### Main Screen

```
┌───────────────────────────────┐
│ 🔥 IMPORTANT & IMMEDIATE      │
├───────────────────────────────┤
│ [ ] Fix prod outage           │
│ [ ] Renew SSL cert            │
│ [x] Pay critical invoice      │
└───────────────────────────────┘

[a] Add  [d] Done  [e] Edit  [q] Quit
```

### Navigation

| Key | Action          |
| --- | --------------- |
| ↑ ↓ | Navigate        |
| a   | Add task        |
| d   | Mark done       |
| x   | Delete          |
| tab | Switch quadrant |
| q   | Quit            |

---

## CLI Commands (Optional)

```bash
actnow add "Fix prod outage" --important --urgent
actnow list --quadrant iim
actnow done <task-id>
```

---

## File Structure

```
actnow/
├── cmd/
│   └── actnow/
│       └── main.go
├── internal/
│   ├── model/
│   │   └── task.go
│   ├── store/
│   │   └── store.go
│   ├── ui/
│   │   └── tui.go
│   └── engine/
│       └── quadrant.go
├── agent.md
├── go.mod
└── README.md
```

---

## Agent Responsibilities

You MUST:

1. Scaffold the project
2. Implement the data model
3. Build the TUI
4. Implement persistence
5. Enforce quadrant rules
6. Provide keyboard-driven UX
7. Write clean, idiomatic Go
8. Add comments where logic is non-trivial

You MUST NOT:

* Add cloud sync
* Add authentication
* Over-engineer

---

## Quality Bar

* App launches in < 100ms
* No crashes on malformed data
* Works fully offline
* Handles at least 5,000 tasks

---

## Testing

* Unit test quadrant logic
* Manual test TUI flows

---

## Deliverables

* Fully working terminal app
* `README.md` with usage
* `agent.md` (this file)

---

## Success Criteria

The app is considered successful when:

* A user can list and act on **Important & Immediate** tasks in under 5 seconds
* Keyboard-only operation is possible
* The app becomes a daily driver for urgent work

---

## Motto

> "What is important and immediate must be done now — without friction."
