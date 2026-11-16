package debloat

// CategoryID identifies one of the main menu categories.
type CategoryID string

const (
	CategoryBloatware     CategoryID = "bloatware"
	CategoryPrivacy       CategoryID = "privacy"
	CategoryOptimizations CategoryID = "optimizations"
)

// StepType defines whether a step disables/uninstalls an Android package or runs a raw shell command.
type StepType int

const (
	StepDisablePackage StepType = iota
	StepShellCommand
)

// CommandStep represents a single atomic operation executed on the Fire Stick.
type CommandStep struct {
	Type               StepType
	Target             string // Package name (for StepDisablePackage) or shell command (for StepShellCommand)
	Description        string
	RestoreTarget      string // Optional shell command to revert StepShellCommand in Restore Mode
	RestoreDescription string // Optional description shown when running in Restore Mode
}

// Action represents a selectable menu item in the TUI that groups related CommandSteps.
type Action struct {
	ID          string
	Category    CategoryID
	Title       string
	Description string
	Steps       []CommandStep
}

// Category groups multiple Actions under a human-readable section header.
type Category struct {
	ID          CategoryID
	Title       string
	Description string
	Actions     []Action
}
