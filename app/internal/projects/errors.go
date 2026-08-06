package projects

import "fmt"

// ErrDuplicate is returned by Service.Add when a project with the same
// normalized name already exists, in ANY state (EC-03 — a wider scope
// than items' ErrDuplicate, which only tracks the offending box). The
// httpapi layer maps it to 409 duplicate_project.
type ErrDuplicate struct{}

func (e ErrDuplicate) Error() string {
	return "project already exists"
}

// ErrNotFound is returned by Service.ChangeState when the target id does
// not exist (EC-12) — httpapi maps it to 404 not_found. Service.Delete
// never returns it (idempotent, EC-13).
type ErrNotFound struct {
	ID int64
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("project %d not found", e.ID)
}

// ErrValidation is returned by Service.Add when the raw name, budget or
// target_date fail validation (EC-01/EC-02/EC-16/EC-17). Code distinguishes
// the reason so httpapi/frontend can render the exact message.
type ErrValidation struct {
	Code    string
	Message string
}

func (e ErrValidation) Error() string {
	return e.Message
}

// Validation error codes.
const (
	ValidationEmpty         = "empty"
	ValidationTooLong       = "too_long"
	ValidationControlChars  = "control_chars"
	ValidationBudgetTooLong = "budget_too_long"
	ValidationInvalidState  = "invalid_state"
	ValidationInvalidDate   = "invalid_target_date"
	// ValidationURLInvalid is a defensive fallback for validateProjectURL
	// (NIU-11) — ideas.ValidateURL only ever returns *ideas.ErrValidation
	// today, so this code is not expected to surface in practice, but
	// validateProjectURL must not panic if that ever changes.
	ValidationURLInvalid = "invalid_url"
)
