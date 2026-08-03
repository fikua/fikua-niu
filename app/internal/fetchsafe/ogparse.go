// ogparse.go implements T-04 (design.md ADR-04): Open Graph metadata
// extraction using golang.org/x/net/html — the same golang.org/x umbrella
// already trusted in go.mod (golang.org/x/text), not a third-party
// scraping dependency. It tokenizes the HTML stream already capped by the
// 2MiB LimitReader (T-03g), stops as soon as </head> is seen (Open Graph
// tags never live in <body>), and extracts og:title/og:image/
// og:description from <meta property="og:*"> tags.
package fetchsafe

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// parseOpenGraph tokenizes r (already limited by io.LimitReader, T-03g)
// and extracts Open Graph metadata from <meta property="og:*"> tags found
// before </head>. Malformed HTML or a complete absence of recognized OG
// tags is treated as "not found" (EC-05) — not as a parser crash — and
// returns a Preview with Partial=true and every field empty, letting the
// caller decide (typically: preview_status='failed', since ideas.Service
// treats zero recovered fields the same as any other fallback).
//
// If the LimitReader cuts the content before </head> is reached (T-03g),
// whatever fields were already found up to that point are still
// returned, marked Partial — this is deliberately a fallback outcome, not
// a fatal error of the fetch itself.
func parseOpenGraph(r io.Reader) (Preview, error) {
	tokenizer := html.NewTokenizer(r)

	var preview Preview
	found := 0

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			// io.EOF (natural end) or a LimitReader cutoff, or malformed
			// markup — in every case, treat as "no more to read", not a
			// crash (EC-05).
			return finishPreview(preview, found), nil

		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()

			if token.Data == "head" && tokenType == html.SelfClosingTagToken {
				// An empty <head/> — nothing to extract.
				return finishPreview(preview, found), nil
			}

			if token.Data == "meta" {
				if applyMetaToken(token, &preview) {
					found++
				}
			}

		case html.EndTagToken:
			token := tokenizer.Token()
			if token.Data == "head" {
				// Open Graph tags never live in <body> — stop reading as
				// soon as </head> closes (ADR-04), avoiding the cost of
				// parsing the rest of a real page.
				return finishPreview(preview, found), nil
			}
		}
	}
}

// applyMetaToken inspects a single <meta> tag's attributes for
// property="og:title|og:image|og:description" + content="...", writing
// into preview when found. Returns true if a recognized OG field was
// set.
func applyMetaToken(token html.Token, preview *Preview) bool {
	var property, content string
	for _, attr := range token.Attr {
		switch strings.ToLower(attr.Key) {
		case "property":
			property = strings.ToLower(strings.TrimSpace(attr.Val))
		case "content":
			content = attr.Val
		}
	}

	if content == "" {
		return false
	}

	switch property {
	case "og:title":
		if preview.Title == "" {
			preview.Title = content
			return true
		}
	case "og:image":
		if preview.ImageURL == "" {
			preview.ImageURL = content
			return true
		}
	case "og:description":
		if preview.Description == "" {
			preview.Description = content
			return true
		}
	}
	return false
}

// finishPreview marks Partial when fewer than all three recognized OG
// fields were found (EC-05) — including the "found nothing at all" case,
// which the caller (ideas.Service) treats as a full fallback.
func finishPreview(preview Preview, found int) Preview {
	preview.Partial = found < 3
	return preview
}
