package engine

import (
	"testing"
	"time"

	"github.com/mrbooshehri/actNow/internal/model"
)

func TestQuadrant(t *testing.T) {
	cases := []struct {
		task model.Task
		want string
	}{
		{task: model.Task{Important: true, Urgent: true}, want: QuadrantImportantImmediate},
		{task: model.Task{Important: true, Urgent: false}, want: QuadrantImportantNotImmediate},
		{task: model.Task{Important: false, Urgent: true}, want: QuadrantNotImportantImmediate},
		{task: model.Task{Important: false, Urgent: false}, want: QuadrantNotImportantNot},
	}

	for _, tc := range cases {
		if got := Quadrant(tc.task); got != tc.want {
			t.Fatalf("expected %s, got %s", tc.want, got)
		}
	}
}

func TestApplyUrgency(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	soon := now.Add(23 * time.Hour)
	late := now.Add(48 * time.Hour)

	urgent := ApplyUrgency(model.Task{DueAt: &soon, Urgent: false}, now)
	if !urgent.Urgent {
		t.Fatalf("expected task to become urgent")
	}

	notUrgent := ApplyUrgency(model.Task{DueAt: &late, Urgent: false}, now)
	if notUrgent.Urgent {
		t.Fatalf("expected task to remain not urgent")
	}
}
