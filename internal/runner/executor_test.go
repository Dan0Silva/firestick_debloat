package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/v0rtx/firestick_debloat/internal/debloat"
)

type mockShell struct {
	responses map[string]string
	commands  []string
}

func (m *mockShell) RunShellCommand(_ context.Context, cmd string) (string, error) {
	m.commands = append(m.commands, cmd)
	if out, ok := m.responses[cmd]; ok {
		return out, nil
	}
	return "", nil
}

func TestExecutorHandlesDisableFallbackWhitelistAndShell(t *testing.T) {
	mock := &mockShell{
		responses: map[string]string{
			// 1. Package disabled directly via pm disable-user
			"pm disable-user --user 0 com.amazon.mp3": "Package com.amazon.mp3 new state: disabled-user",

			// 2. Package refuses disable-user, succeeds on fallback pm uninstall -k --user 0
			"pm disable-user --user 0 com.amazon.tahoe":  "SecurityException: cannot disable package",
			"pm uninstall -k --user 0 com.amazon.tahoe": "Success",

			// 3. Package not present on this FireOS version -> Skipped gracefully
			"pm disable-user --user 0 com.amazon.hedwig": "Failure [not installed for 0]",

			// 4. System optimization command -> Success
			"settings put global window_animation_scale 0.5": "",
		},
	}

	actions := []debloat.Action{
		{
			ID:    "test_action",
			Title: "Test Execution Flow",
			Steps: []debloat.CommandStep{
				{Type: debloat.StepDisablePackage, Target: "com.amazon.mp3", Description: "Disable Music"},
				{Type: debloat.StepDisablePackage, Target: "com.amazon.tahoe", Description: "Disable Kids"},
				{Type: debloat.StepDisablePackage, Target: "com.amazon.hedwig", Description: "Disable Missing Pkg"},
				{Type: debloat.StepDisablePackage, Target: "com.amazon.avod", Description: "Attempt Prime Video (Whitelisted)"},
				{Type: debloat.StepShellCommand, Target: "settings put global window_animation_scale 0.5", Description: "Set Animation 0.5x"},
			},
		},
	}

	exec := NewExecutor(mock)
	var callbackResults []StepResult
	exec.OnStepResult = func(res StepResult) {
		callbackResults = append(callbackResults, res)
	}

	summary := exec.ExecuteActions(context.Background(), actions)

	if len(callbackResults) != 5 {
		t.Fatalf("expected 5 step results in callback, got %d", len(callbackResults))
	}
	if summary.SuccessCount != 3 {
		t.Errorf("expected 3 successes (mp3, tahoe fallback, settings), got %d", summary.SuccessCount)
	}
	if summary.SkippedCount != 2 {
		t.Errorf("expected 2 skipped (hedwig missing, avod whitelisted), got %d", summary.SkippedCount)
	}
	if summary.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", summary.ErrorCount)
	}

	// Verify Prime Video (com.amazon.avod) never triggered any ADB shell command
	for _, cmd := range mock.commands {
		if strings.Contains(cmd, "com.amazon.avod") {
			t.Fatalf("whitelist breach: command %q was sent to shell for protected package com.amazon.avod", cmd)
		}
	}
}

func TestExecutorRestoreMode(t *testing.T) {
	mock := &mockShell{
		responses: map[string]string{
			"pm enable --user 0 com.amazon.mp3":                         "Package com.amazon.mp3 new state: enabled",
			"pm enable --user 0 com.amazon.tahoe":                       "Failure [not installed for 0]",
			"cmd package install-existing --user 0 com.amazon.tahoe":    "Package com.amazon.tahoe installed for user: 0",
			"settings put global window_animation_scale 1.0":            "",
		},
	}

	actions := []debloat.Action{
		{
			ID:    "test_restore",
			Title: "Test Restore Flow",
			Steps: []debloat.CommandStep{
				{Type: debloat.StepDisablePackage, Target: "com.amazon.mp3", Description: "Disable Music", RestoreDescription: "Re-enable Music"},
				{Type: debloat.StepDisablePackage, Target: "com.amazon.tahoe", Description: "Disable Kids", RestoreDescription: "Re-enable Kids"},
				{
					Type:               debloat.StepShellCommand,
					Target:             "settings put global window_animation_scale 0.5",
					Description:        "Set Animation 0.5x",
					RestoreTarget:      "settings put global window_animation_scale 1.0",
					RestoreDescription: "Restore Animation 1.0x",
				},
				{
					Type:               debloat.StepShellCommand,
					Target:             "pm trim-caches 999999999999999999",
					Description:        "Trim Caches",
					RestoreTarget:      "",
					RestoreDescription: "Cache trim does not require restoration",
				},
			},
		},
	}

	exec := NewExecutor(mock)
	exec.RestoreMode = true
	summary := exec.ExecuteActions(context.Background(), actions)

	if summary.SuccessCount != 3 {
		t.Errorf("expected 3 restored successes, got %d", summary.SuccessCount)
	}
	if summary.SkippedCount != 1 {
		t.Errorf("expected 1 skipped (cache trim), got %d", summary.SkippedCount)
	}
	if summary.ErrorCount != 0 {
		t.Errorf("expected 0 errors in restore mode, got %d", summary.ErrorCount)
	}
}

func TestRepairPrimeVideoAndPhotos(t *testing.T) {
	mock := &mockShell{
		responses: map[string]string{
			"pm enable --user 0 com.amazon.firebat": "Package com.amazon.firebat new state: enabled",
			"pm enable --user 0 com.amazon.avod":    "Package com.amazon.avod new state: enabled",
		},
	}

	exec := NewExecutor(mock)
	summary := exec.RepairPrimeVideoAndPhotos(context.Background())
	if summary.SuccessCount < 2 {
		t.Fatalf("expected at least 2 re-enabled Prime Video packages, got %d", summary.SuccessCount)
	}
}
