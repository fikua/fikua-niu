package items

import "fmt"

// ErrDuplicate is returned by Service.Add when an active item with the
// same normalized name already exists, in either box (EC-06). The
// httpapi layer maps it to 409 duplicate_item and names the box in the
// user-facing message (proposal.md §8.4.3).
type ErrDuplicate struct {
	ExistingLocation Location
}

func (e ErrDuplicate) Error() string {
	return fmt.Sprintf("item already exists in %s", e.ExistingLocation)
}

// ErrNotFound is returned by Service.Move when the target id does not
// exist (EC-12) — httpapi maps it to 404 not_found. Service.Delete never
// returns it (idempotent, EC-11).
type ErrNotFound struct {
	ID int64
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("item %d not found", e.ID)
}

// ErrValidation is returned by Service.Add when the raw name fails
// validation (EC-01/EC-02/EC-03/EC-05). Code distinguishes the reason so
// httpapi/frontend can render the exact message from proposal.md §8.4.3.
type ErrValidation struct {
	Code    string
	Message string
}

func (e ErrValidation) Error() string {
	return e.Message
}

// Validation error codes (EC-01/EC-02/EC-03/EC-05).
const (
	ValidationEmpty        = "empty"
	ValidationTooLong      = "too_long"
	ValidationControlChars = "control_chars"
)
