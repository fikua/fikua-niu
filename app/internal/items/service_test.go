package items

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

// T-24 — validation boundaries (EC-01..EC-05), Unicode corpus (EC-04,
// NFR-11) and the NFC vs NFD case documented in ADR-02.

func TestNormalizeName_TrimAndLowercase(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Llet", "llet"},
		{"padded", "  LLET  ", "llet"},
		{"catalan accents", "Pastanaga", "pastanaga"},
		{"catalan uppercase accents", "PASTANAGA", "pastanaga"},
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

// TestNormalizeName_NFCvsNFD verifies the exact scenario documented in
// ADR-02: composed "Àrab" (À = 1 code point) and decomposed "Àrab" (A +
// combining acute accent) look identical on screen but compare unequal
// under ToLower alone. After NormalizeName (NFC -> trim -> lower), they
// must be equal.
func TestNormalizeName_NFCvsNFD(t *testing.T) {
	composed := "Àrab"                      // NFC: À is U+00C0, single code point
	decomposed := norm.NFD.String(composed) // decompose to A + combining grave

	if composed == decomposed {
		t.Fatalf("test setup invalid: composed and decomposed forms are byte-identical")
	}
	if len(composed) == len(decomposed) {
		t.Fatalf("test setup invalid: expected different byte lengths (composed=%d, decomposed=%d)", len(composed), len(decomposed))
	}

	// Sanity check the failure mode this test guards against: plain
	// ToLower (no NFC) must NOT consider these equal.
	if strings.ToLower(composed) == strings.ToLower(decomposed) {
		t.Fatalf("test setup invalid: ToLower alone already treats composed/decomposed as equal — the ADR-02 scenario cannot be demonstrated")
	}

	gotComposed := NormalizeName(composed)
	gotDecomposed := NormalizeName(decomposed)
	if gotComposed != gotDecomposed {
		t.Errorf("NormalizeName must fold NFC/NFD to the same value: got %q vs %q", gotComposed, gotDecomposed)
	}
}

func TestService_Add_EmptyOrWhitespace(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo, repo)

	cases := []string{"", "   ", "\t\t", "\n"}
	for _, raw := range cases {
		_, err := svc.Add(t.Context(), 1, raw)
		var valErr ErrValidation
		if err == nil {
			t.Fatalf("Add(%q) = nil error, want ErrValidation", raw)
		}
		if !errorsAs(err, &valErr) || valErr.Code != ValidationEmpty {
			t.Fatalf("Add(%q) = %v, want ErrValidation{Code: empty}", raw, err)
		}
	}
}

func TestService_Add_LengthBoundary(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo, repo)

	name200 := strings.Repeat("a", 200)
	item, err := svc.Add(t.Context(), 1, name200)
	if err != nil {
		t.Fatalf("Add(200 chars) unexpected error: %v", err)
	}
	if item.Name != name200 {
		t.Errorf("Add(200 chars) stored name mismatch")
	}

	name201 := strings.Repeat("b", 201)
	_, err = svc.Add(t.Context(), 1, name201)
	var valErr ErrValidation
	if !errorsAs(err, &valErr) || valErr.Code != ValidationTooLong {
		t.Fatalf("Add(201 chars) = %v, want ErrValidation{Code: too_long}", err)
	}
}

func TestService_Add_ControlChars(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo, repo)

	_, err := svc.Add(t.Context(), 1, "Llet\x01amb control")
	var valErr ErrValidation
	if !errorsAs(err, &valErr) || valErr.Code != ValidationControlChars {
		t.Fatalf("Add(control chars) = %v, want ErrValidation{Code: control_chars}", err)
	}
}

func TestService_Add_UnicodeCorpus(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo, repo)

	names := []string{"Pastanagues 🥕", "O'Neill", "Formatge d'ovella", "Àrab", "Maçã"}
	for _, name := range names {
		item, err := svc.Add(t.Context(), 1, name)
		if err != nil {
			t.Fatalf("Add(%q) unexpected error: %v", name, err)
		}
		if item.Name != name {
			t.Errorf("Add(%q) round-trip mismatch: got %q", name, item.Name)
		}
	}
}

func TestService_Add_DuplicateAcrossBoxes(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, repo, repo)

	if _, err := svc.Add(t.Context(), 1, "Llet"); err != nil {
		t.Fatalf("seed Add unexpected error: %v", err)
	}

	dupCases := []string{"llet", "Llet ", "LLET", "  llet  "}
	for _, raw := range dupCases {
		_, err := svc.Add(t.Context(), 1, raw)
		var dupErr ErrDuplicate
		if !errorsAs(err, &dupErr) {
			t.Fatalf("Add(%q) = %v, want ErrDuplicate", raw, err)
		}
	}
}

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
