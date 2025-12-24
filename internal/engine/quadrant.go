package engine

import (
	"time"

	"github.com/mrbooshehri/actNow/internal/model"
)

const (
	QuadrantImportantImmediate    = "Important & Immediate"
	QuadrantImportantNotImmediate = "Important & Not Immediate"
	QuadrantNotImportantImmediate = "Not Important & Immediate"
	QuadrantNotImportantNot       = "Not Important & Not Immediate"
)

func Quadrant(t model.Task) string {
	switch {
	case t.Important && t.Urgent:
		return QuadrantImportantImmediate
	case t.Important:
		return QuadrantImportantNotImmediate
	case t.Urgent:
		return QuadrantNotImportantImmediate
	default:
		return QuadrantNotImportantNot
	}
}

func QuadrantIndex(t model.Task) int {
	switch Quadrant(t) {
	case QuadrantImportantImmediate:
		return 0
	case QuadrantImportantNotImmediate:
		return 1
	case QuadrantNotImportantImmediate:
		return 2
	default:
		return 3
	}
}

func ApplyUrgency(t model.Task, now time.Time) model.Task {
	if t.DueAt == nil {
		return t
	}
	if t.DueAt.Sub(now) <= 24*time.Hour {
		t.Urgent = true
	}
	return t
}
