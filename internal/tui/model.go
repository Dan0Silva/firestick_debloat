package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/v0rtx/firestick_debloat/internal/debloat"
	"github.com/v0rtx/firestick_debloat/internal/runner"
)

// AppState represents the current screen of the TUI state machine.
type AppState int

const (
	StateConnecting AppState = iota
	StateMenu
	StateRunning
	StateSummary
)

// DeviceConnector abstracts the ADB connection and shell execution lifecycle.
type DeviceConnector interface {
	runner.ShellRunner
	Address() string
	DeviceBanner() string
	Connect(ctx context.Context) error
	Close() error
}

// Menu cursor index helpers:
//   - cursor == 0: "[ ] Select All Actions"
//   - cursor == 1 .. len(actions): individual debloat.Action items
//   - cursor == len(actions) + 1: "[ ▶ EXECUTE SELECTED ACTIONS ]"

// Internal Bubble Tea messages
type connSuccessMsg struct {
	banner string
}

type connErrorMsg struct {
	err error
}

type tvPromptMsg struct{}

type stepProgressMsg struct {
	result runner.StepResult
}

type executionDoneMsg struct {
	summary runner.Summary
}

type tickMsg time.Time

// Model holds the complete state for the Bubble Tea application.
type Model struct {
	state           AppState
	device          DeviceConnector
	categories      []debloat.Category
	actions         []debloat.Action
	selected        map[string]bool
	cursor          int
	restoreMode     bool
	waitingTVPrompt bool
	spinnerFrame    int
	connErr         error
	validationErr   string

	// Execution progress state
	totalSteps int
	liveLogs   []runner.StepResult
	summary    runner.Summary
	eventCh    chan tea.Msg
}

// NewModel constructs the initial TUI Model for the given DeviceConnector and catalog.
func NewModel(device DeviceConnector, categories []debloat.Category) Model {
	var flat []debloat.Action
	for _, cat := range categories {
		flat = append(flat, cat.Actions...)
	}

	return Model{
		state:      StateConnecting,
		device:     device,
		categories: categories,
		actions:    flat,
		selected:   make(map[string]bool, len(flat)),
		cursor:     0,
		eventCh:    make(chan tea.Msg, 64),
	}
}

// SetRestoreMode configures whether the TUI starts in Restore Mode (--restore).
func (m *Model) SetRestoreMode(restore bool) {
	m.restoreMode = restore
}

// RestoreMode returns true if the TUI is currently in Restore/Undo mode.
func (m Model) RestoreMode() bool {
	return m.restoreMode
}

// NotifyTVPrompt can be wired to adb.Client.PromptHandler so the Connecting screen
// informs the user to check their TV and accept the USB/ADB debugging dialog.
func (m *Model) NotifyTVPrompt() {
	select {
	case m.eventCh <- tvPromptMsg{}:
	default:
	}
}

// Init starts the initial connection to the Fire Stick, the spinner ticker, and the async event listener.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		connectDeviceCmd(m.device),
		waitForEventCmd(m.eventCh),
		tickCmd(),
	)
}

func connectDeviceCmd(device DeviceConnector) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := device.Connect(ctx); err != nil {
			return connErrorMsg{err: err}
		}
		return connSuccessMsg{banner: device.DeviceBanner()}
	}
}

func waitForEventCmd(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// maxCursorIndex returns the index of the Execute button at the bottom of the menu.
func (m Model) maxCursorIndex() int {
	return len(m.actions) + 1
}

// executeButtonIndex returns the cursor position corresponding to the Execute button.
func (m Model) executeButtonIndex() int {
	return len(m.actions) + 1
}

// allSelected returns true if every action in the catalog is currently checked.
func (m Model) allSelected() bool {
	if len(m.actions) == 0 {
		return false
	}
	for _, act := range m.actions {
		if !m.selected[act.ID] {
			return false
		}
	}
	return true
}

// selectedActions returns the ordered slice of Actions currently checked by the user.
func (m Model) selectedActions() []debloat.Action {
	var chosen []debloat.Action
	for _, act := range m.actions {
		if m.selected[act.ID] {
			chosen = append(chosen, act)
		}
	}
	return chosen
}
