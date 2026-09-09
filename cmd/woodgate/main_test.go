package main

import (
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodgate/internal/backgroundjobs"
	"github.com/woodleighschool/woodgate/internal/config"
)

func TestBackgroundJobsAreAbsentWhenEntraIsDisabled(t *testing.T) {
	runtime, syncJobs, err := newBackgroundJobs(config.Config{}, nil, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newBackgroundJobs: %v", err)
	}
	if runtime != nil {
		t.Fatal("runtime is configured without any enabled jobs")
	}
	status, err := syncJobs.Status(t.Context())
	if err != nil {
		t.Fatalf("sync status: %v", err)
	}
	if status.Enabled || status.Activity != backgroundjobs.ActivityIdle {
		t.Fatalf("sync status = %+v, want disabled and idle", status)
	}
}
