package tui

import (
	"fmt"
	"strings"

	"github.com/v0rtx/firestick_debloat/internal/runner"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// View renders the current screen of the TUI state machine.
func (m Model) View() string {
	var content string

	switch m.state {
	case StateConnecting:
		content = m.viewConnecting()
	case StateMenu:
		content = m.viewMenu()
	case StateRunning:
		content = m.viewRunning()
	case StateSummary:
		content = m.viewSummary()
	}

	return "\n" + boxStyle.Render(content) + "\n"
}

func (m Model) viewConnecting() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("🔥 FireStick Debloat — Pure Go Edition"))
	b.WriteString("\n\n")

	if m.connErr != nil {
		b.WriteString(badgeErrStyle.Render(fmt.Sprintf("✖ Connection failed: %v", m.connErr)))
		b.WriteString("\n\n")
		b.WriteString(normalItemStyle.Render("Tips:\n"))
		b.WriteString(normalItemStyle.Render("  1. Verify the Fire Stick IP address is correct and on the same network.\n"))
		b.WriteString(normalItemStyle.Render("  2. Ensure Settings > My Fire TV > Developer Options > ADB Debugging is ON.\n"))
		b.WriteString(normalItemStyle.Render("  3. If prompted on your TV screen, check 'Always allow' and click OK."))
		return b.String()
	}

	spin := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	b.WriteString(subHeaderStyle.Render(fmt.Sprintf("%s Connecting to Fire Stick at %s...", spin, m.device.Address())))
	b.WriteString("\n\n")

	if m.waitingTVPrompt {
		b.WriteString(badgeSkipStyle.Render("📺 ACTION REQUIRED ON YOUR TV:"))
		b.WriteString("\n")
		b.WriteString(selectedItemStyle.Render("   Look at your TV screen, check 'Always allow from this computer', and press OK!"))
	} else {
		b.WriteString(normalItemStyle.Render("Establishing native ADB over TCP handshake (:5555)..."))
	}

	b.WriteString("\n\n")
	b.WriteString(footerHelpStyle.Render("Press q or Ctrl+C to cancel"))
	return b.String()
}

func (m Model) viewMenu() string {
	var b strings.Builder

	bannerInfo := m.device.Address()
	if banner := m.device.DeviceBanner(); banner != "" {
		bannerInfo = fmt.Sprintf("%s (%s)", m.device.Address(), banner)
	}

	modeBadge := headerStyle.Render("🔥 DEBLOAT MODE")
	actionPrefix := "Disable/Optimize: "
	if m.restoreMode {
		modeBadge = subHeaderStyle.Render("♻️  RESTORE MODE (Re-enable Packages & Defaults)")
		actionPrefix = "Restore: "
	}

	b.WriteString(fmt.Sprintf("%s — Connected to %s\n", modeBadge, selectedItemStyle.Render(bannerInfo)))
	b.WriteString(normalItemStyle.Render("🛡️  Protected by Whitelist: Amazon Prime Video & Amazon Photos"))
	b.WriteString("\n\n")

	// Index 0: Select All toggle
	selectAllCursor := "  "
	if m.cursor == 0 {
		selectAllCursor = cursorStyle.Render("> ")
	}
	selectAllBox := uncheckedBoxStyle.Render("[ ]")
	if m.allSelected() {
		selectAllBox = checkedBoxStyle.Render("[x]")
	}
	selectAllLabel := "Select All Actions (Marcar Tudo)"
	if m.cursor == 0 {
		selectAllLabel = selectedItemStyle.Render(selectAllLabel)
	} else {
		selectAllLabel = normalItemStyle.Render(selectAllLabel)
	}
	b.WriteString(fmt.Sprintf("%s%s %s\n", selectAllCursor, selectAllBox, selectAllLabel))

	// Render categories and their actions (indices 1 .. len(actions))
	actionIdx := 1
	for _, cat := range m.categories {
		b.WriteString(categoryHeaderStyle.Render(fmt.Sprintf("── %s ──", cat.Title)))
		b.WriteString("\n")

		for _, act := range cat.Actions {
			cur := "  "
			isFocused := m.cursor == actionIdx
			if isFocused {
				cur = cursorStyle.Render("> ")
			}

			box := uncheckedBoxStyle.Render("[ ]")
			if m.selected[act.ID] {
				box = checkedBoxStyle.Render("[x]")
			}

			displayTitle := actionPrefix + act.Title
			title := normalItemStyle.Render(displayTitle)
			if isFocused {
				title = selectedItemStyle.Render(displayTitle)
			}

			b.WriteString(fmt.Sprintf("%s%s %s\n", cur, box, title))
			actionIdx++
		}
	}

	// Render contextual description for currently highlighted item
	b.WriteString("\n")
	b.WriteString(descriptionBoxStyle.Render("ℹ " + m.currentItemDescription()))
	b.WriteString("\n\n")

	// Render validation message if user clicked Execute with 0 items selected
	if m.validationErr != "" {
		b.WriteString(badgeErrStyle.Render("⚠ " + m.validationErr))
		b.WriteString("\n\n")
	}

	// Index len(actions)+1: Execute button
	execFocused := m.cursor == m.executeButtonIndex()
	btnVerb := "EXECUTE SELECTED ACTIONS"
	if m.restoreMode {
		btnVerb = "RESTORE SELECTED ACTIONS"
	}
	btnLabel := fmt.Sprintf("▶ %s (%d selected)", btnVerb, len(m.selectedActions()))
	if execFocused {
		b.WriteString(cursorStyle.Render("> ") + executeBtnActiveStyle.Render(btnLabel))
	} else {
		b.WriteString("  " + executeBtnIdleStyle.Render(btnLabel))
	}

	b.WriteString("\n")
	b.WriteString(footerHelpStyle.Render("↑/↓: Navigate • Space: Toggle • a: Select All • r: Toggle Restore Mode • Enter: Execute • q: Quit"))
	return b.String()
}

func (m Model) currentItemDescription() string {
	if m.cursor == 0 {
		return "Toggle all debloat, privacy, and system optimization actions at once."
	}
	if m.cursor == m.executeButtonIndex() {
		if m.restoreMode {
			return "Re-enable all checked packages and restore default settings on your Fire Stick."
		}
		return "Apply all checked optimizations and bloatware removals to your Fire Stick now."
	}
	idx := m.cursor - 1
	if idx >= 0 && idx < len(m.actions) {
		return m.actions[idx].Description
	}
	return ""
}

func (m Model) viewRunning() string {
	var b strings.Builder

	spin := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	completed := len(m.liveLogs)
	verb := "Applying Optimizations"
	if m.restoreMode {
		verb = "Restoring Packages & Settings"
	}
	b.WriteString(headerStyle.Render(fmt.Sprintf("%s %s on %s...", spin, verb, m.device.Address())))
	b.WriteString("\n\n")

	// Render progress bar
	b.WriteString(renderProgressBar(completed, m.totalSteps, 44))
	b.WriteString(fmt.Sprintf(" %d/%d steps\n\n", completed, m.totalSteps))

	// Show recent live logs (up to last 12 lines so terminal never overflows)
	start := 0
	if len(m.liveLogs) > 12 {
		start = len(m.liveLogs) - 12
	}
	for _, res := range m.liveLogs[start:] {
		b.WriteString(formatStepResultLine(res) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(footerHelpStyle.Render("Please wait while commands are executed over ADB..."))
	return b.String()
}

func (m Model) viewSummary() string {
	var b strings.Builder

	title := "✨ FireStick Debloat — Execution Complete!"
	if m.restoreMode {
		title = "♻️  FireStick Debloat — Restoration Complete!"
	}
	b.WriteString(headerStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf(
		"Summary:  %s   %s   %s\n\n",
		badgeOKStyle.Render(fmt.Sprintf("✔ %d Succeeded", m.summary.SuccessCount)),
		badgeSkipStyle.Render(fmt.Sprintf("↷ %d Skipped", m.summary.SkippedCount)),
		badgeErrStyle.Render(fmt.Sprintf("✖ %d Errors", m.summary.ErrorCount)),
	))

	// Show up to last 15 results
	start := 0
	if len(m.summary.Results) > 15 {
		start = len(m.summary.Results) - 15
		b.WriteString(normalItemStyle.Render(fmt.Sprintf("... (%d earlier steps omitted) ...\n", start)))
	}
	for _, res := range m.summary.Results[start:] {
		b.WriteString(formatStepResultLine(res) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(subHeaderStyle.Render("🛡️  Prime Video and Amazon Photos remain untouched and ready to use."))
	b.WriteString("\n")
	b.WriteString(footerHelpStyle.Render("Press Enter, Space, or q to exit"))
	return b.String()
}

func formatStepResultLine(res runner.StepResult) string {
	var badge string
	switch res.Status {
	case runner.StatusSuccess:
		badge = badgeOKStyle.Render("[ OK ]")
	case runner.StatusSkipped:
		badge = badgeSkipStyle.Render("[SKIP]")
	case runner.StatusError:
		badge = badgeErrStyle.Render("[ERR ]")
	}
	return fmt.Sprintf("  %s %s — %s", badge, selectedItemStyle.Render(res.Step.Description), normalItemStyle.Render(res.Message))
}

func renderProgressBar(current, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if current > total {
		current = total
	}
	filled := (current * width) / total
	empty := width - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return subHeaderStyle.Render("[" + bar + "]")
}
