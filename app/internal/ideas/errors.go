package ideas

import "fmt"

// ErrNotFound is returned by Repository.Get when the target id does not
// exist. Service exposes no Get method of its own — today the only
// caller of Repository.Get is IdeasRepository.Create's internal
// re-fetch of the row it just inserted (internal/store/ideas.go), so
// this error is not currently reachable from any HTTP handler (F-02,
// review.md). Kept as forward-looking API surface for a future
// Service.Get, not removed. Service.Delete never returns it — deletion
// is idempotent by design (EC-15) and never calls Get.
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
