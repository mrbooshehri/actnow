package model

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"time"
)

const (
	StatusPending  = "pending"
	StatusDone     = "done"
	StatusDeferred = "deferred"
)

type Task struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Important      bool       `json:"important"`
	Urgent         bool       `json:"urgent"`
	DueAt          *time.Time `json:"due_at,omitempty"`
	Impact         string     `json:"impact,omitempty"`
	NextAction     string     `json:"next_action,omitempty"`
	PlannedDate    *time.Time `json:"planned_date,omitempty"`
	DelegateTo     string     `json:"delegate_to,omitempty"`
	DeleteReason   string     `json:"delete_reason,omitempty"`
	EffortEstimate string     `json:"effort_estimate,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (t Task) IsDone() bool {
	return t.Status == StatusDone
}

func NewTask(title, description string, important, urgent bool, dueAt *time.Time) Task {
	return Task{
		ID:          newID(),
		Title:       title,
		Description: description,
		Important:   important,
		Urgent:      urgent,
		DueAt:       dueAt,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
	}
}

func newID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "")
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]), "=")
}
