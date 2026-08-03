package ideas

import "fmt"

// ErrNotFound is returned by Service.Get when the target id does not
// exist. Service.Delete never returns it (idempotent, EC-15).
type ErrNotFound struct {
	ID int64
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("idea %d not found", e.ID)
}

// ErrValidation is returned by Service.Add when the raw URL fails format
// or scheme validation (EC-01/EC-10). Code distinguishes the reason so
// httpapi/frontend can render the exact message — but never distinguishes
// "destination forbidden" from any other fallback reason (NFR-06,
// design.md §6.1: that rejection never surfaces synchronously, since the
// scrape happens after the 201 response, ADR-03).
type ErrValidation struct {
	Code    string
	Message string
}

func (e ErrValidation) Error() string {
	return e.Message
}

// Validation error codes (EC-01/EC-10).
const (
	ValidationEmpty          = "empty"
	ValidationInvalidFormat  = "invalid_format"
	ValidationSchemeRejected = "scheme_rejected"
)
