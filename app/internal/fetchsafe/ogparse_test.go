package fetchsafe

import (
	"strings"
	"testing"
)

// F-09/F-06 — og:image scheme validation: a value recovered from the
// fetched page's own content must never reach the caller unfiltered.
// javascript:/data:/file:/scheme-relative values are silently discarded
// (field stays empty), never surfaced as a parser error.

func TestParseOpenGraph_ImageJavascriptScheme_Discarded(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:title" content="Amb imatge maliciosa">
<meta property="og:image" content="javascript:alert(document.cookie)">
<meta property="og:description" content="desc">
</head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty (javascript: scheme must be discarded)", preview.ImageURL)
	}
	if !preview.Partial {
		t.Error("preview.Partial = false, want true (image field was discarded, only 2/3 found)")
	}
}

func TestParseOpenGraph_ImageDataScheme_Discarded(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:image" content="data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==">
</head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty (data: scheme must be discarded)", preview.ImageURL)
	}
}

func TestParseOpenGraph_ImageFileScheme_Discarded(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:image" content="file:///etc/passwd">
</head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty (file: scheme must be discarded)", preview.ImageURL)
	}
}

func TestParseOpenGraph_ImageSchemeRelative_Discarded(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:image" content="//evil.example/x.png">
</head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty (scheme-relative URL must be discarded)", preview.ImageURL)
	}
}

func TestParseOpenGraph_ImageHTTPAndHTTPSSchemes_Accepted(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		htmlDoc := `<html><head>
<meta property="og:image" content="` + scheme + `://example.com/img.jpg">
</head><body></body></html>`

		preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
		if err != nil {
			t.Fatalf("parseOpenGraph: %v", err)
		}
		want := scheme + "://example.com/img.jpg"
		if preview.ImageURL != want {
			t.Errorf("scheme %q: ImageURL = %q, want %q", scheme, preview.ImageURL, want)
		}
	}
}

// F-10/F-07 — recovered OG fields are truncated, never rejected outright.

func TestParseOpenGraph_TitleExceedsCap_Truncated(t *testing.T) {
	longTitle := strings.Repeat("a", 500)
	htmlDoc := `<html><head><meta property="og:title" content="` + longTitle + `"></head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if len(preview.Title) != 300 {
		t.Errorf("Title length = %d, want 300 (truncated)", len(preview.Title))
	}
}

func TestParseOpenGraph_DescriptionExceedsCap_Truncated(t *testing.T) {
	longDesc := strings.Repeat("b", 2000)
	htmlDoc := `<html><head><meta property="og:description" content="` + longDesc + `"></head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if len(preview.Description) != 1000 {
		t.Errorf("Description length = %d, want 1000 (truncated)", len(preview.Description))
	}
}

func TestParseOpenGraph_ImageURLExceedsCap_Truncated(t *testing.T) {
	longPath := strings.Repeat("c", 3000)
	htmlDoc := `<html><head><meta property="og:image" content="https://example.com/` + longPath + `"></head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if len(preview.ImageURL) != 2048 {
		t.Errorf("ImageURL length = %d, want 2048 (truncated)", len(preview.ImageURL))
	}
}

func TestParseOpenGraph_FieldsWithinCap_Unmodified(t *testing.T) {
	htmlDoc := `<html><head>
<meta property="og:title" content="Un títol normal">
<meta property="og:image" content="https://example.com/img.jpg">
<meta property="og:description" content="Una descripció normal.">
</head><body></body></html>`

	preview, err := parseOpenGraph(strings.NewReader(htmlDoc))
	if err != nil {
		t.Fatalf("parseOpenGraph: %v", err)
	}
	if preview.Title != "Un títol normal" {
		t.Errorf("Title = %q, want unmodified", preview.Title)
	}
	if preview.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("ImageURL = %q, want unmodified", preview.ImageURL)
	}
	if preview.Description != "Una descripció normal." {
		t.Errorf("Description = %q, want unmodified", preview.Description)
	}
}

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
