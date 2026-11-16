package runner

import "github.com/v0rtx/firestick_debloat/internal/debloat"

// StepStatus represents the outcome of executing a single CommandStep.
type StepStatus string

const (
	StatusSuccess StepStatus = "OK"
	StatusSkipped StepStatus = "SKIP"
	StatusError   StepStatus = "ERR"
)

// StepResult holds the execution details of a single CommandStep.
type StepResult struct {
	ActionID    string
	ActionTitle string
	Step        debloat.CommandStep
	Status      StepStatus
	Message     string
}

// Summary aggregates the final execution counts across all selected actions.
type Summary struct {
	Results      []StepResult
	SuccessCount int
	SkippedCount int
	ErrorCount   int
}

// Add appends a StepResult and updates the corresponding counters.
func (s *Summary) Add(res StepResult) {
	s.Results = append(s.Results, res)
	switch res.Status {
	case StatusSuccess:
		s.SuccessCount++
	case StatusSkipped:
		s.SkippedCount++
	case StatusError:
		s.ErrorCount++
	}
}
