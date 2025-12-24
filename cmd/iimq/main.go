package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mrbooshehri/actNow/internal/model"
	"github.com/mrbooshehri/actNow/internal/store"
	"github.com/mrbooshehri/actNow/internal/ui"
)

func main() {
	st, err := store.NewStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize store: %v\n", err)
		os.Exit(1)
	}

	data, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load tasks: %v\n", err)
		os.Exit(1)
	}

	var (
		tasks        []model.Task
		corruptFound bool
	)
	if err := store.DecodeTasks(data, &tasks); err != nil {
		if err == store.ErrCorruptData {
			fmt.Fprintf(os.Stderr, "task file appears corrupted; starting with empty list\n")
			tasks = []model.Task{}
			corruptFound = true
		} else {
			fmt.Fprintf(os.Stderr, "failed to decode tasks: %v\n", err)
			os.Exit(1)
		}
	}

	for i := range tasks {
		if tasks[i].Status == "" {
			tasks[i].Status = model.StatusPending
		}
		if tasks[i].CreatedAt.IsZero() {
			tasks[i].CreatedAt = time.Now()
		}
	}

	m := ui.New(st, tasks)
	if corruptFound {
		m.SetStatus("Corrupt data detected; started empty", true)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running program: %v\n", err)
		os.Exit(1)
	}
}
