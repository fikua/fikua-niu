package projects

import (
	"context"
	"errors"
	"testing"
	"time"

	"niu/internal/ideas"
)

// T-11 — NIU-11's link preview: Add with a url enqueues and resolves;
// Add without a url enqueues nothing and leaves preview_status NULL;
// a fetch error resolves to failed; a partial result with zero recovered
// fields also resolves to failed. Follows internal/ideas/service_test.go
// exactly (tasks.md T-11), including its fake repo + polling pattern.

// newTestServiceWithFetch wires a Service against a fresh fakeRepo and
// the given fetch — mirrors ideas.newTestService.
func newTestServiceWithFetch(t *testing.T, fetch PreviewFetcher) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	pool := ideas.NewWorkerPool(context.Background())
	t.Cleanup(pool.Close)
	return NewService(repo, repo, fetch, pool), repo
}

// waitForPreviewStatus polls the fake repo until the project reaches a
// non-nil, non-pending PreviewStatus or the timeout elapses — the worker
// pool resolves asynchronously by design, so tests cannot assert on the
// state synchronously after Add returns.
func waitForPreviewStatus(t *testing.T, repo *fakeRepo, id int64, timeout time.Duration) Project {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		p, err := repo.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("Get(%d): %v", id, err)
		}
		if p.PreviewStatus != nil && *p.PreviewStatus != PreviewPending {
			return p
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("project %d still pending after %s", id, timeout)
	return Project{}
}

func TestService_Add_WithURL_EnqueuesAndResolvesToReady(t *testing.T) {
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Title: "T", ImageURL: "https://example.com/i.jpg", Description: "D"}, nil
	})

	created, err := svc.Add(t.Context(), 1, "Televisor 4K", nil, nil, strPtr("https://example.com/tv"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if created.PreviewStatus == nil || *created.PreviewStatus != PreviewPending {
		t.Fatalf("Add returned PreviewStatus = %v, want pending (201-equivalent never waits for the scrape)", created.PreviewStatus)
	}
	if created.Title != nil || created.ImageURL != nil {
		t.Fatalf("Add returned non-nil preview fields for a pending project: %+v", created)
	}

	resolved := waitForPreviewStatus(t, repo, created.ID, time.Second)
	if resolved.PreviewStatus == nil || *resolved.PreviewStatus != PreviewReady {
		t.Fatalf("resolved PreviewStatus = %v, want ready", resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "T" {
		t.Errorf("Title = %v, want T", resolved.Title)
	}
	if resolved.ImageURL == nil || *resolved.ImageURL != "https://example.com/i.jpg" {
		t.Errorf("ImageURL = %v", resolved.ImageURL)
	}
	if resolved.Description == nil || *resolved.Description != "D" {
		t.Errorf("Description = %v, want D (persisted even though the DTO never exposes it, T-05)", resolved.Description)
	}
}

func TestService_Add_WithoutURL_NeverEnqueuesAndPreviewStatusStaysNil(t *testing.T) {
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		t.Fatal("fetch must not be called when Add was given no url")
		return Preview{}, nil
	})

	created, err := svc.Add(t.Context(), 1, "Estanteria", nil, nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if created.PreviewStatus != nil {
		t.Fatalf("Add(no url) PreviewStatus = %v, want nil (never 'pending' — there is no preview to resolve)", created.PreviewStatus)
	}
	if created.URL != nil {
		t.Fatalf("Add(no url) URL = %v, want nil", created.URL)
	}

	// Give any (incorrectly) enqueued job a chance to run before asserting
	// the status never moved off nil.
	time.Sleep(20 * time.Millisecond)
	stored, err := repo.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.PreviewStatus != nil {
		t.Fatalf("stored PreviewStatus = %v, want nil to stay nil", stored.PreviewStatus)
	}
}

func TestService_Add_EmptyURLTreatedAsNoURL(t *testing.T) {
	// decision 2 (tasks.md context): a whitespace-only url is the same as
	// omitting it entirely — not a validation error.
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		t.Fatal("fetch must not be called for a blank url")
		return Preview{}, nil
	})

	created, err := svc.Add(t.Context(), 1, "Terrassa", nil, nil, strPtr("   "))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if created.URL != nil || created.PreviewStatus != nil {
		t.Fatalf("Add(whitespace url) = %+v, want nil URL and nil PreviewStatus", created)
	}

	list, _ := repo.List(t.Context())
	if len(list) != 1 {
		t.Fatalf("List() = %d project(s), want 1", len(list))
	}
}

func TestService_Add_RejectedURLScheme_NoRowCreated(t *testing.T) {
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		t.Fatal("fetch must not be called for a rejected scheme")
		return Preview{}, nil
	})

	_, err := svc.Add(t.Context(), 1, "Rentadora", nil, nil, strPtr("javascript:alert(1)"))
	var val ErrValidation
	if !errorsAs(err, &val) {
		t.Fatalf("Add(bad scheme) = %v, want ErrValidation", err)
	}

	list, _ := repo.List(t.Context())
	if len(list) != 0 {
		t.Fatalf("Add(bad scheme) created %d row(s), want 0", len(list))
	}
}

func TestService_Add_FetchError_ResolvesToFailed_NeverBlocksAdd(t *testing.T) {
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{}, errors.New("simulated: destination forbidden")
	})

	created, err := svc.Add(t.Context(), 1, "Sofà", nil, nil, strPtr("https://example.com/blocked"))
	if err != nil {
		t.Fatalf("Add must never fail because of a scrape outcome: %v", err)
	}

	resolved := waitForPreviewStatus(t, repo, created.ID, time.Second)
	if resolved.PreviewStatus == nil || *resolved.PreviewStatus != PreviewFailed {
		t.Fatalf("resolved PreviewStatus = %v, want failed", resolved.PreviewStatus)
	}
	if resolved.Title != nil || resolved.ImageURL != nil || resolved.Description != nil {
		t.Errorf("expected all-nil preview fields on failure, got %+v", resolved)
	}
}

func TestService_Add_PartialWithZeroFieldsRecovered_TreatedAsFailed(t *testing.T) {
	// A "partial" result carrying literally nothing recovered must render
	// as fallback, not as an empty "complete" card — same rule as
	// ideas.Service.resolvePreview.
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Partial: true}, nil
	})

	created, err := svc.Add(t.Context(), 1, "Nevera", nil, nil, strPtr("https://example.com/nevera"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	resolved := waitForPreviewStatus(t, repo, created.ID, time.Second)
	if resolved.PreviewStatus == nil || *resolved.PreviewStatus != PreviewFailed {
		t.Fatalf("resolved PreviewStatus = %v, want failed (nothing recovered)", resolved.PreviewStatus)
	}
}

func TestService_Add_PartialWithSomeFieldsRecovered_ResolvesToPartial(t *testing.T) {
	svc, repo := newTestServiceWithFetch(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Title: "Only title", Partial: true}, nil
	})

	created, err := svc.Add(t.Context(), 1, "Cadires noves", nil, nil, strPtr("https://example.com/cadires"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	resolved := waitForPreviewStatus(t, repo, created.ID, time.Second)
	if resolved.PreviewStatus == nil || *resolved.PreviewStatus != PreviewPartial {
		t.Fatalf("resolved PreviewStatus = %v, want partial", resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "Only title" {
		t.Errorf("Title = %v, want %q", resolved.Title, "Only title")
	}
	if resolved.ImageURL != nil || resolved.Description != nil {
		t.Errorf("expected nil ImageURL/Description, got %+v", resolved)
	}
}
