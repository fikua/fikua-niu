package fetchsafe

import (
	"strings"
	"testing"
)

// T-21 — AC-01/AC-03/EC-05: Open Graph parsing unit tests. Full tags
// (success), partial tags (some fields missing), and no/malformed tags
// (treated as "not found", never a parser crash).

func TestParseOpenGraph_FullTags(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:title" content="Un restaurant fantàstic">
<meta property="og:image" content="https://example.com/img.jpg">
<meta property="og:description" content="La millor paella de la ciutat.">
</head><body>ignored</body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.Partial {
		t.Error("preview.Partial = true, want false (all three OG tags present)")
	}
	if preview.Title != "Un restaurant fantàstic" {
		t.Errorf("Title = %q", preview.Title)
	}
	if preview.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("ImageURL = %q", preview.ImageURL)
	}
	if preview.Description != "La millor paella de la ciutat." {
		t.Errorf("Description = %q", preview.Description)
	}
}

func TestParseOpenGraph_PartialTags(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:title" content="Només títol">
</head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if !preview.Partial {
		t.Error("preview.Partial = false, want true (only og:title present)")
	}
	if preview.Title != "Només títol" {
		t.Errorf("Title = %q", preview.Title)
	}
	if preview.ImageURL != "" || preview.Description != "" {
		t.Errorf("expected empty ImageURL/Description, got %+v", preview)
	}
}

func TestParseOpenGraph_NoOGTags_TreatedAsNotFound(t *testing.T) {
	htmlDoc := `<html><head><title>Plain page</title></head><body>hello</body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if !preview.Partial {
		t.Error("preview.Partial = false, want true (no OG tags at all)")
	}
	if preview.Title != "" || preview.ImageURL != "" || preview.Description != "" {
		t.Errorf("expected all-empty preview, got %+v", preview)
	}
}

func TestParseOpenGraph_MalformedHTML_DoesNotCrash(t *testing.T) {
	malformed := `<html <head oops="` + "\x00" + `"><meta property=og:title content=Broken`

	preview, err := parseOpenGraph(strings.NewReader(malformed))
	if err != nil {
		t.Fatalf("parseOpenGraph on malformed input returned error (should degrade gracefully): %v", err)
	}
	_ = preview // no crash is the assertion; content extracted (if any) is a bonus
}

func TestParseOpenGraph_StopsAtHeadClose_IgnoresBodyMetaTags(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:title" content="Head title">
</head><body>
<meta property="og:description" content="Should be ignored">
</body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.Description != "" {
		t.Errorf("Description = %q, want empty (body meta tags must be ignored)", preview.Description)
	}
}
