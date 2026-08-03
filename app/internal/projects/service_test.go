package projects

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

// T-22 — name/budget/target_date validation boundaries (EC-01, EC-02,
// EC-16, EC-17). T-23 — normalization/duplicate rejection across states
// (ADR-02, EC-03).

func errorsAs(err error, target any) bool {
	switch t := target.(type) {
	case *ErrValidation:
		if v, ok := err.(ErrValidation); ok {
			*t = v
			return true
		}
	case *ErrDuplicate:
		if v, ok := err.(ErrDuplicate); ok {
			*t = v
			return true
		}
	case *ErrNotFound:
		if v, ok := err.(ErrNotFound); ok {
			*t = v
			return true
		}
	}
	return false
}

func strPtr(s string) *string { return &s }

func TestService_Add_EmptyOrWhitespaceName(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	cases := []string{"", "   ", "\t\t", "\n"}
	for _, raw := range cases {
		_, err := svc.Add(t.Context(), 1, raw, nil, nil)
		var valErr ErrValidation
		if !errorsAs(err, &valErr) || valErr.Code != ValidationEmpty {
			t.Fatalf("Add(%q) = %v, want ErrValidation{Code: empty}", raw, err)
		}
	}
}

// EC-02: name at the 200/201 character boundary.
func TestService_Add_NameLengthBoundary(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	name200 := strings.Repeat("a", 200)
	project, err := svc.Add(t.Context(), 1, name200, nil, nil)
	if err != nil {
		t.Fatalf("Add(200 chars) unexpected error: %v", err)
	}
	if project.Name != name200 {
		t.Errorf("Add(200 chars) stored name mismatch")
	}

	name201 := strings.Repeat("b", 201)
	_, err = svc.Add(t.Context(), 1, name201, nil, nil)
	var valErr ErrValidation
	if !errorsAs(err, &valErr) || valErr.Code != ValidationTooLong {
		t.Fatalf("Add(201 chars) = %v, want ErrValidation{Code: too_long}", err)
	}
}

// EC-16: budget at the 200/201 character boundary.
func TestService_Add_BudgetLengthBoundary(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	budget200 := strings.Repeat("c", 200)
	project, err := svc.Add(t.Context(), 1, "Televisor", strPtr(budget200), nil)
	if err != nil {
		t.Fatalf("Add(budget 200 chars) unexpected error: %v", err)
	}
	if project.Budget == nil || *project.Budget != budget200 {
		t.Errorf("Add(budget 200 chars) stored budget mismatch")
	}

	budget201 := strings.Repeat("d", 201)
	_, err = svc.Add(t.Context(), 1, "Nevera", strPtr(budget201), nil)
	var valErr ErrValidation
	if !errorsAs(err, &valErr) || valErr.Code != ValidationBudgetTooLong {
		t.Fatalf("Add(budget 201 chars) = %v, want ErrValidation{Code: budget_too_long}", err)
	}
}

// AC-14: budget is optional — nil and empty-string both accepted, and
// the resulting project shows no budget field.
func TestService_Add_BudgetOptional(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	p1, err := svc.Add(t.Context(), 1, "Sofà", nil, nil)
	if err != nil {
		t.Fatalf("Add(nil budget) unexpected error: %v", err)
	}
	if p1.Budget != nil {
		t.Errorf("Add(nil budget) = %+v, want nil budget", p1.Budget)
	}

	p2, err := svc.Add(t.Context(), 1, "Taula", strPtr("   "), nil)
	if err != nil {
		t.Fatalf("Add(whitespace-only budget) unexpected error: %v", err)
	}
	if p2.Budget != nil {
		t.Errorf("Add(whitespace-only budget) = %+v, want nil budget", p2.Budget)
	}
}

// EC-17: a target_date in the past is accepted without error.
func TestService_Add_TargetDatePastAccepted(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	project, err := svc.Add(t.Context(), 1, "Pintar habitació", nil, strPtr("2000-01-01"))
	if err != nil {
		t.Fatalf("Add(past target_date) unexpected error: %v, want acceptance (EC-17)", err)
	}
	if project.TargetDate == nil || *project.TargetDate != "2000-01-01" {
		t.Errorf("Add(past target_date) stored value mismatch: %+v", project.TargetDate)
	}
}

// Format validation: an invalid target_date is rejected.
func TestService_Add_TargetDateInvalidFormat(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	cases := []string{"not-a-date", "2026-13-01", "01/12/2026", "2026-12-1"}
	for _, raw := range cases {
		_, err := svc.Add(t.Context(), 1, "Projecte", nil, strPtr(raw))
		var valErr ErrValidation
		if !errorsAs(err, &valErr) || valErr.Code != ValidationInvalidDate {
			t.Fatalf("Add(target_date=%q) = %v, want ErrValidation{Code: invalid_target_date}", raw, err)
		}
	}
}

// AC-15: target_date is optional — nil and empty-string both accepted.
func TestService_Add_TargetDateOptional(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	p1, err := svc.Add(t.Context(), 1, "Terrassa", nil, nil)
	if err != nil {
		t.Fatalf("Add(nil target_date) unexpected error: %v", err)
	}
	if p1.TargetDate != nil {
		t.Errorf("Add(nil target_date) = %+v, want nil", p1.TargetDate)
	}

	p2, err := svc.Add(t.Context(), 1, "Jardí", nil, strPtr(""))
	if err != nil {
		t.Fatalf("Add(empty target_date) unexpected error: %v", err)
	}
	if p2.TargetDate != nil {
		t.Errorf("Add(empty target_date) = %+v, want nil", p2.TargetDate)
	}
}

func TestNormalizeName_TrimAndLowercase(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Televisor", "televisor"},
		{"padded", "  Televisor Nou  ", "televisor nou"},
		{"catalan accents", "Pintar l'habitació", "pintar l'habitació"},
		{"catalan uppercase accents", "PINTAR L'HABITACIÓ", "pintar l'habitació"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeName(tc.in)
			if got != tc.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestNormalizeName_NFCvsNFD confirms reuse of items.NormalizeName
// (ADR-02): composed and decomposed Unicode forms fold to the same value.
func TestNormalizeName_NFCvsNFD(t *testing.T) {
	composed := "Àrab"
	decomposed := norm.NFD.String(composed)

	if composed == decomposed {
		t.Fatalf("test setup invalid: composed and decomposed forms are byte-identical")
	}

	gotComposed := NormalizeName(composed)
	gotDecomposed := NormalizeName(decomposed)
	if gotComposed != gotDecomposed {
		t.Errorf("NormalizeName must fold NFC/NFD to the same value: got %q vs %q", gotComposed, gotDecomposed)
	}
}

// EC-03: a duplicate is rejected regardless of casing/whitespace, using
// the same combinations already proved for NIU-1.
func TestService_Add_DuplicateTrimmedCaseInsensitive(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	if _, err := svc.Add(t.Context(), 1, "Televisor nou", nil, nil); err != nil {
		t.Fatalf("seed Add unexpected error: %v", err)
	}

	dupCases := []string{"televisor nou", "Televisor nou ", "TELEVISOR NOU", "  televisor nou  "}
	for _, raw := range dupCases {
		_, err := svc.Add(t.Context(), 1, raw, nil, nil)
		var dupErr ErrDuplicate
		if !errorsAs(err, &dupErr) {
			t.Fatalf("Add(%q) = %v, want ErrDuplicate", raw, err)
		}
	}
}

// EC-03: a duplicate is rejected regardless of the existing project's
// state — idea, decidit AND fet all block a new duplicate. This is the
// wider-scope behaviour ADR-02 documents versus NIU-1 (which only checks
// active items across two boxes).
func TestService_Add_DuplicateAcrossAllStates(t *testing.T) {
	states := []State{StateIdea, StateDecidit, StateFet}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			repo := newFakeRepo()
			svc := NewService(repo, repo)

			created, err := svc.Add(t.Context(), 1, "Repintar cuina", nil, nil)
			if err != nil {
				t.Fatalf("seed Add unexpected error: %v", err)
			}

			if state != StateIdea {
				if _, err := svc.ChangeState(t.Context(), 1, created.ID, state); err != nil {
					t.Fatalf("seed ChangeState(%s) unexpected error: %v", state, err)
				}
			}

			_, err = svc.Add(t.Context(), 1, "repintar cuina", nil, nil)
			var dupErr ErrDuplicate
			if !errorsAs(err, &dupErr) {
				t.Fatalf("Add(duplicate) while existing is %s = %v, want ErrDuplicate", state, err)
			}
		})
	}
}

// AC-09: every direction between the three states is a valid move,
// including reversions (decidit -> idea, fet -> decidit).
func TestService_ChangeState_AnyDirectionValid(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	created, err := svc.Add(t.Context(), 1, "Moble nou", nil, nil)
	if err != nil {
		t.Fatalf("seed Add unexpected error: %v", err)
	}

	transitions := []State{StateDecidit, StateFet, StateDecidit, StateIdea, StateFet, StateIdea}
	for _, next := range transitions {
		updated, err := svc.ChangeState(t.Context(), 2, created.ID, next)
		if err != nil {
			t.Fatalf("ChangeState(%s) unexpected error: %v", next, err)
		}
		if updated.State != next {
			t.Fatalf("ChangeState(%s) resulting state = %q, want %q", next, updated.State, next)
		}
	}
}

func TestService_ChangeState_InvalidStateRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	created, err := svc.Add(t.Context(), 1, "Cadires noves", nil, nil)
	if err != nil {
		t.Fatalf("seed Add unexpected error: %v", err)
	}

	_, err = svc.ChangeState(t.Context(), 1, created.ID, State("abandonat"))
	var valErr ErrValidation
	if !errorsAs(err, &valErr) || valErr.Code != ValidationInvalidState {
		t.Fatalf("ChangeState(abandonat) = %v, want ErrValidation{Code: invalid_state} (EC-06: no such state exists)", err)
	}
}

func TestService_ChangeState_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	_, err := svc.ChangeState(t.Context(), 1, 9999, StateDecidit)
	var notFound ErrNotFound
	if !errorsAs(err, &notFound) {
		t.Fatalf("ChangeState(nonexistent id) = %v, want ErrNotFound", err)
	}
}

func TestService_Delete_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	created, err := svc.Add(t.Context(), 1, "Estanteria", nil, nil)
	if err != nil {
		t.Fatalf("seed Add unexpected error: %v", err)
	}

	if err := svc.Delete(t.Context(), 1, created.ID); err != nil {
		t.Fatalf("first Delete unexpected error: %v", err)
	}
	if err := svc.Delete(t.Context(), 1, created.ID); err != nil {
		t.Fatalf("second Delete unexpected error: %v, want idempotent success (EC-13)", err)
	}
}

// EC-04: after deleting a project, the same name is accepted as a new
// project — the duplicate check only looks at active (non-deleted) rows.
func TestService_Add_DuplicateAllowedAfterDelete(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo)

	created, err := svc.Add(t.Context(), 1, "Rentadora", nil, nil)
	if err != nil {
		t.Fatalf("seed Add unexpected error: %v", err)
	}
	if err := svc.Delete(t.Context(), 1, created.ID); err != nil {
		t.Fatalf("Delete unexpected error: %v", err)
	}

	recreated, err := svc.Add(t.Context(), 1, "Rentadora", nil, nil)
	if err != nil {
		t.Fatalf("Add(same name after delete) = %v, want success (EC-04)", err)
	}
	if recreated.Name != "Rentadora" {
		t.Errorf("recreated project name = %q, want Rentadora", recreated.Name)
	}
}
