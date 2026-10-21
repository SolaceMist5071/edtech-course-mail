package main

import (
	"testing"
	"time"
)

func TestReminderWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		delta time.Duration
		want  bool
	}{{"inside", 24 * time.Hour, true}, {"outside", 72 * time.Hour, false}, {"past", -time.Hour, false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reminderNeeded(now, now.Add(tt.delta)); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
