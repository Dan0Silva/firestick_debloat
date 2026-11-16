package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/v0rtx/firestick_debloat/internal/debloat"
)

// ShellRunner abstracts the ADB client's shell command execution so Executor can be
// easily unit-tested without a physical device.
type ShellRunner interface {
	RunShellCommand(ctx context.Context, cmd string) (string, error)
}

// Executor runs selected debloat/optimization (or restore) actions sequentially on the Fire Stick.
type Executor struct {
	shell        ShellRunner
	RestoreMode  bool             // When true, re-enables packages and restores default settings
	OnStepResult func(StepResult) // Optional callback invoked after each step completes
}

// NewExecutor creates a new Executor bound to the given ShellRunner (e.g. *adb.Client).
func NewExecutor(shell ShellRunner) *Executor {
	return &Executor{shell: shell}
}

// RepairPrimeVideoAndPhotos actively re-enables Amazon Prime Video (com.amazon.avod,
// com.amazon.firebat, com.amazon.hedwig, and its required shared libraries) and Amazon Photos
// (com.amazon.bueller.photos) on the Fire Stick.
func (e *Executor) RepairPrimeVideoAndPhotos(ctx context.Context) Summary {
	repairAction := debloat.Action{
		ID:    "repair_prime_video_photos",
		Title: "Repair & Ensure Prime Video & Amazon Photos",
	}
	for _, pkg := range debloat.PrimeVideoRepairPackages {
		repairAction.Steps = append(repairAction.Steps, debloat.CommandStep{
			Type:               debloat.StepDisablePackage,
			Target:             pkg,
			Description:        fmt.Sprintf("Ensure %s is enabled", pkg),
			RestoreDescription: fmt.Sprintf("Ensure %s is enabled", pkg),
		})
	}

	var summary Summary
	for _, step := range repairAction.Steps {
		if ctx.Err() != nil {
			break
		}
		res := e.enablePackage(ctx, repairAction, step)
		summary.Add(res)
		if e.OnStepResult != nil {
			e.OnStepResult(res)
		}
	}
	return summary
}

// ExecuteActions runs all steps for the provided list of Actions and returns a Summary.
func (e *Executor) ExecuteActions(ctx context.Context, actions []debloat.Action) Summary {
	var summary Summary

	for _, action := range actions {
		for _, step := range action.Steps {
			if ctx.Err() != nil {
				res := StepResult{
					ActionID:    action.ID,
					ActionTitle: action.Title,
					Step:        step,
					Status:      StatusError,
					Message:     fmt.Sprintf("execution canceled: %v", ctx.Err()),
				}
				summary.Add(res)
				if e.OnStepResult != nil {
					e.OnStepResult(res)
				}
				return summary
			}

			var res StepResult
			if e.RestoreMode {
				res = e.restoreStep(ctx, action, step)
			} else {
				res = e.executeStep(ctx, action, step)
			}

			summary.Add(res)
			if e.OnStepResult != nil {
				e.OnStepResult(res)
			}
		}
	}

	return summary
}

func (e *Executor) executeStep(ctx context.Context, action debloat.Action, step debloat.CommandStep) StepResult {
	switch step.Type {
	case debloat.StepDisablePackage:
		return e.disablePackage(ctx, action, step)
	case debloat.StepShellCommand:
		return e.runShellStep(ctx, action, step, step.Target, step.Description)
	default:
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusError,
			Message:     fmt.Sprintf("unknown step type: %d", step.Type),
		}
	}
}

func (e *Executor) restoreStep(ctx context.Context, action debloat.Action, step debloat.CommandStep) StepResult {
	desc := step.RestoreDescription
	if desc == "" {
		desc = "Restore " + step.Description
	}
	stepCopy := step
	stepCopy.Description = desc

	switch step.Type {
	case debloat.StepDisablePackage:
		return e.enablePackage(ctx, action, stepCopy)
	case debloat.StepShellCommand:
		if strings.TrimSpace(step.RestoreTarget) == "" {
			return StepResult{
				ActionID:    action.ID,
				ActionTitle: action.Title,
				Step:        stepCopy,
				Status:      StatusSkipped,
				Message:     "no restore needed for this one-time action",
			}
		}
		return e.runShellStep(ctx, action, stepCopy, step.RestoreTarget, desc)
	default:
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        stepCopy,
			Status:      StatusError,
			Message:     fmt.Sprintf("unknown step type: %d", step.Type),
		}
	}
}

func (e *Executor) disablePackage(ctx context.Context, action debloat.Action, step debloat.CommandStep) StepResult {
	pkg := strings.TrimSpace(step.Target)

	// Defense-in-depth: never allow disabling a whitelisted package (Prime Video, Photos, Core OS)
	if reason, protected := debloat.WhitelistReason(pkg); protected {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSkipped,
			Message:     fmt.Sprintf("protected by whitelist (%s)", reason),
		}
	}

	// Primary method: pm disable-user --user 0 <pkg>
	disableCmd := fmt.Sprintf("pm disable-user --user 0 %s", pkg)
	out, err := e.shell.RunShellCommand(ctx, disableCmd)
	if err != nil {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusError,
			Message:     fmt.Sprintf("ADB error on %s: %v", pkg, err),
		}
	}

	lowerOut := strings.ToLower(out)
	if strings.Contains(lowerOut, "disabled-user") || strings.Contains(lowerOut, "new state: disabled") {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSuccess,
			Message:     fmt.Sprintf("disabled %s", pkg),
		}
	}

	if isPackageMissingOutput(lowerOut) {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSkipped,
			Message:     fmt.Sprintf("%s not installed on this device", pkg),
		}
	}

	// Fallback method: pm uninstall -k --user 0 <pkg>
	uninstallCmd := fmt.Sprintf("pm uninstall -k --user 0 %s", pkg)
	unOut, unErr := e.shell.RunShellCommand(ctx, uninstallCmd)
	if unErr != nil {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusError,
			Message:     fmt.Sprintf("ADB error uninstalling %s: %v", pkg, unErr),
		}
	}

	lowerUnOut := strings.ToLower(unOut)
	if strings.Contains(lowerUnOut, "success") {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSuccess,
			Message:     fmt.Sprintf("uninstalled (user 0) %s", pkg),
		}
	}

	if isPackageMissingOutput(lowerUnOut) {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSkipped,
			Message:     fmt.Sprintf("%s already removed or not installed", pkg),
		}
	}

	return StepResult{
		ActionID:    action.ID,
		ActionTitle: action.Title,
		Step:        step,
		Status:      StatusError,
		Message:     fmt.Sprintf("failed to disable %s: %s", pkg, firstLine(unOut, out)),
	}
}

func (e *Executor) enablePackage(ctx context.Context, action debloat.Action, step debloat.CommandStep) StepResult {
	pkg := strings.TrimSpace(step.Target)

	// Step 1: If package was uninstalled for user 0, reinstall it first via cmd package install-existing
	installCmd := fmt.Sprintf("cmd package install-existing --user 0 %s", pkg)
	instOut, _ := e.shell.RunShellCommand(ctx, installCmd)

	// Step 2: Ensure package is enabled via pm enable --user 0 <pkg>
	enableCmd := fmt.Sprintf("pm enable --user 0 %s", pkg)
	out, err := e.shell.RunShellCommand(ctx, enableCmd)
	if err != nil {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusError,
			Message:     fmt.Sprintf("ADB error enabling %s: %v", pkg, err),
		}
	}

	lowerOut := strings.ToLower(out)
	lowerInst := strings.ToLower(instOut)

	if strings.Contains(lowerOut, "new state: enabled") ||
		strings.Contains(lowerInst, "installed for user") ||
		strings.Contains(lowerInst, "success") {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSuccess,
			Message:     fmt.Sprintf("re-enabled %s", pkg),
		}
	}

	if isPackageMissingOutput(lowerOut) || isPackageMissingOutput(lowerInst) || strings.Contains(lowerInst, "doesn't exist") {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusSkipped,
			Message:     fmt.Sprintf("%s not present on this firmware", pkg),
		}
	}

	return StepResult{
		ActionID:    action.ID,
		ActionTitle: action.Title,
		Step:        step,
		Status:      StatusError,
		Message:     fmt.Sprintf("failed to restore %s: %s", pkg, firstLine(out, instOut)),
	}
}

func (e *Executor) runShellStep(ctx context.Context, action debloat.Action, step debloat.CommandStep, targetCmd, successMsg string) StepResult {
	cmd := strings.TrimSpace(targetCmd)
	out, err := e.shell.RunShellCommand(ctx, cmd)
	if err != nil {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusError,
			Message:     fmt.Sprintf("ADB error running %q: %v", cmd, err),
		}
	}

	lowerOut := strings.ToLower(out)
	if strings.Contains(lowerOut, "exception") || strings.Contains(lowerOut, "error:") || strings.Contains(lowerOut, "failure") {
		return StepResult{
			ActionID:    action.ID,
			ActionTitle: action.Title,
			Step:        step,
			Status:      StatusError,
			Message:     fmt.Sprintf("command failed: %s", firstLine(out, cmd)),
		}
	}

	return StepResult{
		ActionID:    action.ID,
		ActionTitle: action.Title,
		Step:        step,
		Status:      StatusSuccess,
		Message:     successMsg,
	}
}

func isPackageMissingOutput(lowerOut string) bool {
	return strings.Contains(lowerOut, "unknown package") ||
		strings.Contains(lowerOut, "not installed for 0") ||
		strings.Contains(lowerOut, "package_not_found") ||
		strings.Contains(lowerOut, "nullpointerexception") // older FireOS pm throws NPE when disabling a missing package
}

func firstLine(primary, fallback string) string {
	s := strings.TrimSpace(primary)
	if s == "" {
		s = strings.TrimSpace(fallback)
	}
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}
