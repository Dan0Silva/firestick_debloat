package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/v0rtx/firestick_debloat/internal/debloat"
	"github.com/v0rtx/firestick_debloat/internal/runner"
)

// Update processes incoming Bubble Tea messages and key presses across all TUI states.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.state == StateConnecting || m.state == StateRunning {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, tickCmd()
		}
		return m, nil

	case tvPromptMsg:
		m.waitingTVPrompt = true
		return m, waitForEventCmd(m.eventCh)

	case connSuccessMsg:
		m.state = StateMenu
		return m, nil

	case connErrorMsg:
		m.connErr = msg.err
		return m, tea.Quit

	case stepProgressMsg:
		m.liveLogs = append(m.liveLogs, msg.result)
		return m, waitForEventCmd(m.eventCh)

	case executionDoneMsg:
		m.summary = msg.summary
		m.state = StateSummary
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit keys
	if key == "ctrl+c" || (key == "q" && m.state != StateRunning) {
		if m.device != nil {
			_ = m.device.Close()
		}
		return m, tea.Quit
	}

	switch m.state {
	case StateMenu:
		return m.handleMenuKeys(key)
	case StateSummary:
		if key == "enter" || key == " " || key == "esc" {
			if m.device != nil {
				_ = m.device.Close()
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) handleMenuKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.validationErr = ""
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = m.maxCursorIndex()
		}

	case "down", "j":
		m.validationErr = ""
		if m.cursor < m.maxCursorIndex() {
			m.cursor++
		} else {
			m.cursor = 0
		}

	case "a", "A":
		m.validationErr = ""
		m.toggleAll()

	case "r", "R":
		m.validationErr = ""
		m.restoreMode = !m.restoreMode

	case " ":
		m.validationErr = ""
		if m.cursor == 0 {
			m.toggleAll()
		} else if m.cursor == m.executeButtonIndex() {
			return m.startExecution()
		} else {
			m.toggleActionAtCursor()
		}

	case "enter":
		m.validationErr = ""
		if m.cursor == m.executeButtonIndex() {
			return m.startExecution()
		} else if m.cursor == 0 {
			m.toggleAll()
		} else {
			m.toggleActionAtCursor()
		}
	}

	return m, nil
}

func (m *Model) toggleAll() {
	targetState := !m.allSelected()
	for _, act := range m.actions {
		m.selected[act.ID] = targetState
	}
}

func (m *Model) toggleActionAtCursor() {
	idx := m.cursor - 1
	if idx >= 0 && idx < len(m.actions) {
		actID := m.actions[idx].ID
		m.selected[actID] = !m.selected[actID]
	}
}

func (m Model) startExecution() (tea.Model, tea.Cmd) {
	chosen := m.selectedActions()
	if len(chosen) == 0 {
		m.validationErr = "Please select at least one action (or press 'a' to Select All) before executing."
		return m, nil
	}

	total := 0
	for _, act := range chosen {
		total += len(act.Steps)
	}

	m.state = StateRunning
	m.totalSteps = total
	m.liveLogs = nil

	return m, tea.Batch(
		runActionsAsyncCmd(m.device, chosen, m.restoreMode, m.eventCh),
		waitForEventCmd(m.eventCh),
		tickCmd(),
	)
}

func runActionsAsyncCmd(device DeviceConnector, chosen []debloat.Action, restoreMode bool, eventCh chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		exec := runner.NewExecutor(device)

		// Always ensure Prime Video & Amazon Photos packages and shared libraries are enabled
		// on the Fire Stick (heals devices if com.amazon.firebat was previously disabled).
		exec.RepairPrimeVideoAndPhotos(ctx)

		exec.RestoreMode = restoreMode
		exec.OnStepResult = func(res runner.StepResult) {
			eventCh <- stepProgressMsg{result: res}
		}
		summary := exec.ExecuteActions(ctx, chosen)
		return executionDoneMsg{summary: summary}
	}
}
