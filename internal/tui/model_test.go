package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/v0rtx/firestick_debloat/internal/debloat"
)

type fakeDevice struct {
	addr   string
	banner string
}

func (f *fakeDevice) Address() string      { return f.addr }
func (f *fakeDevice) DeviceBanner() string { return f.banner }
func (f *fakeDevice) Connect(_ context.Context) error {
	return nil
}
func (f *fakeDevice) Close() error { return nil }
func (f *fakeDevice) RunShellCommand(_ context.Context, cmd string) (string, error) {
	if strings.HasPrefix(cmd, "pm disable-user") {
		return "new state: disabled-user", nil
	}
	if strings.HasPrefix(cmd, "pm enable") {
		return "new state: enabled", nil
	}
	return "Success", nil
}

func TestMenuNavigationSelectAllRestoreToggleAndExecutionTransition(t *testing.T) {
	dev := &fakeDevice{addr: "192.168.1.100:5555", banner: "AFTT"}
	catalog := debloat.DefaultCatalog()
	m := NewModel(dev, catalog)

	// Transition from Connecting -> Menu
	updated, _ := m.Update(connSuccessMsg{banner: "AFTT"})
	m = updated.(Model)
	if m.state != StateMenu {
		t.Fatalf("expected StateMenu after connSuccessMsg, got %v", m.state)
	}

	// Cursor starts at 0 ("Select All"). Pressing Space should select all actions.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(Model)
	if !m.allSelected() {
		t.Fatalf("expected all actions to be selected after toggling Select All at cursor 0")
	}

	// Pressing 'a' should toggle all actions off
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if m.allSelected() || len(m.selectedActions()) != 0 {
		t.Fatalf("expected 0 selected actions after pressing 'a' to unselect all")
	}

	// Pressing 'r' should toggle Restore Mode on and off
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)
	if !m.RestoreMode() {
		t.Fatalf("expected RestoreMode() == true after pressing 'r'")
	}
	if menuOut := m.View(); !strings.Contains(menuOut, "RESTORE MODE") {
		t.Errorf("expected menu view to indicate RESTORE MODE, got:\n%s", menuOut)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)
	if m.RestoreMode() {
		t.Fatalf("expected RestoreMode() == false after pressing 'r' again")
	}

	// Move down to first action (cursor == 1) and toggle it with Space
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", m.cursor)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(Model)
	if len(m.selectedActions()) != 1 {
		t.Fatalf("expected 1 selected action, got %d", len(m.selectedActions()))
	}

	// Jump to Execute button and press Enter
	m.cursor = m.executeButtonIndex()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state != StateRunning {
		t.Fatalf("expected StateRunning after pressing Enter on Execute button, got %v", m.state)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil batch command when starting execution")
	}

	// Verify View renders non-empty output across states
	if out := m.View(); !strings.Contains(out, "Applying Optimizations") {
		t.Errorf("expected running view to contain header, got:\n%s", out)
	}
}
