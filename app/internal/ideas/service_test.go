package ideas

import (
	"context"
	"errors"
	"testing"
	"time"
)

// T-05/T-06/T-09 — Service.Add validation + immediate pending creation,
// Service.Delete idempotency, worker pool resolution.

// waitForStatus polls the fake repo until the idea reaches one of the
// terminal statuses (ready/partial/failed) or the timeout elapses — the
// worker pool resolves asynchronously, on purpose (ADR-03), so tests
// cannot assert on the state synchronously after Add returns.
func waitForStatus(t *testing.T, repo *fakeRepo, id int64, timeout time.Duration) Idea {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		idea, err := repo.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("Get(%d): %v", id, err)
		}
		if idea.PreviewStatus != PreviewPending {
			return idea
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("idea %d still pending after %s", id, timeout)
	return Idea{}
}

func newTestService(t *testing.T, fetch PreviewFetcher) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	pool := NewWorkerPool(context.Background())
	t.Cleanup(pool.Close)
	return NewService(repo, repo, fetch, pool), repo
}

func TestServiceAdd_EmptyURL_Rejected(t *testing.T) {
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		t.Fatal("fetch must not be called for an invalid URL")
		return Preview{}, nil
	})

	_, err := svc.Add(context.Background(), 1, "   ")
	var val ErrValidation
	if !errors.As(err, &val) || val.Code != ValidationEmpty {
		t.Fatalf("Add(empty) error = %v, want ErrValidation{Code: empty}", err)
	}

	list, _ := repo.List(context.Background())
	if len(list) != 0 {
		t.Fatalf("Add(empty) created %d row(s), want 0", len(list))
	}
}

func TestServiceAdd_NonHTTPScheme_Rejected(t *testing.T) {
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		t.Fatal("fetch must not be called for a rejected scheme")
		return Preview{}, nil
	})

	cases := []string{"file:///etc/passwd", "javascript:alert(1)", "ftp://example.com", "data:text/html,x"}
	for _, raw := range cases {
		_, err := svc.Add(context.Background(), 1, raw)
		var val ErrValidation
		if !errors.As(err, &val) || val.Code != ValidationSchemeRejected {
			t.Errorf("Add(%q) error = %v, want ErrValidation{Code: scheme_rejected}", raw, err)
		}
	}

	list, _ := repo.List(context.Background())
	if len(list) != 0 {
		t.Fatalf("Add(non-http scheme) created %d row(s), want 0", len(list))
	}
}

func TestServiceAdd_ValidURL_CreatesRowImmediatelyPending(t *testing.T) {
	release := make(chan struct{})
	svc, _ := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		<-release // held open until the test explicitly releases it
		return Preview{Title: "Title", ImageURL: "https://example.com/i.jpg", Description: "Desc"}, nil
	})
	defer close(release)

	idea, err := svc.Add(context.Background(), 1, "https://example.com/page")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if idea.PreviewStatus != PreviewPending {
		t.Fatalf("Add returned status = %q, want pending (ADR-03: 201 never waits for the scrape)", idea.PreviewStatus)
	}
	if idea.Title != nil || idea.ImageURL != nil || idea.Description != nil {
		t.Fatalf("Add returned non-nil preview fields for a pending idea: %+v", idea)
	}
}

func TestServiceAdd_SuccessfulScrape_ResolvesToReady(t *testing.T) {
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Title: "T", ImageURL: "https://example.com/i.jpg", Description: "D"}, nil
	})

	idea, err := svc.Add(context.Background(), 1, "https://example.com/page")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	resolved := waitForStatus(t, repo, idea.ID, time.Second)
	if resolved.PreviewStatus != PreviewReady {
		t.Fatalf("resolved status = %q, want ready", resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "T" {
		t.Errorf("Title = %v, want T", resolved.Title)
	}
}

func TestServiceAdd_PartialScrape_ResolvesToPartial(t *testing.T) {
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Title: "Only title", Partial: true}, nil
	})

	idea, err := svc.Add(context.Background(), 1, "https://example.com/page")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	resolved := waitForStatus(t, repo, idea.ID, time.Second)
	if resolved.PreviewStatus != PreviewPartial {
		t.Fatalf("resolved status = %q, want partial", resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "Only title" {
		t.Errorf("Title = %v, want %q", resolved.Title, "Only title")
	}
	if resolved.ImageURL != nil || resolved.Description != nil {
		t.Errorf("expected nil ImageURL/Description, got %+v", resolved)
	}
}

func TestServiceAdd_FetchError_ResolvesToFailed_NeverBlocksAdd(t *testing.T) {
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{}, errors.New("simulated: destination forbidden")
	})

	idea, err := svc.Add(context.Background(), 1, "https://example.com/blocked")
	if err != nil {
		t.Fatalf("Add must never fail because of a scrape outcome (AC-02): %v", err)
	}

	resolved := waitForStatus(t, repo, idea.ID, time.Second)
	if resolved.PreviewStatus != PreviewFailed {
		t.Fatalf("resolved status = %q, want failed", resolved.PreviewStatus)
	}
	if resolved.Title != nil || resolved.ImageURL != nil || resolved.Description != nil {
		t.Errorf("expected all-nil preview fields on failure, got %+v", resolved)
	}
}

func TestServiceAdd_EmptyPartialResult_TreatedAsFailed(t *testing.T) {
	// A "partial" result carrying literally nothing recovered must render
	// as fallback (Estat B), not as an empty "complete" card (Estat C).
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Partial: true}, nil
	})

	idea, err := svc.Add(context.Background(), 1, "https://example.com/page")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	resolved := waitForStatus(t, repo, idea.ID, time.Second)
	if resolved.PreviewStatus != PreviewFailed {
		t.Fatalf("resolved status = %q, want failed (nothing recovered)", resolved.PreviewStatus)
	}
}

func TestServiceDelete_Idempotent(t *testing.T) {
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{}, errors.New("no scrape needed for this test")
	})

	idea, err := svc.Add(context.Background(), 1, "https://example.com/page")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := svc.Delete(context.Background(), 1, idea.ID); err != nil {
		t.Fatalf("first Delete: %v", err)
	}
	if err := svc.Delete(context.Background(), 1, idea.ID); err != nil {
		t.Fatalf("second Delete (already gone) must not error (EC-15): %v", err)
	}

	list, _ := repo.List(context.Background())
	if len(list) != 0 {
		t.Fatalf("idea still present after Delete: %+v", list)
	}

	repo.mu.Lock()
	deleteEvents := 0
	for _, e := range repo.events {
		if e.kind == "idea_deleted" {
			deleteEvents++
		}
	}
	repo.mu.Unlock()
	if deleteEvents != 1 {
		t.Fatalf("idea_deleted events = %d, want exactly 1 (idempotent, EC-15)", deleteEvents)
	}
}

func TestServiceAdd_MultipleConcurrentAdds_AllResolveIndependently(t *testing.T) {
	// EC-16: double submission (or several concurrent adds) must never
	// deadlock the pool nor corrupt state — each resolves on its own.
	svc, repo := newTestService(t, func(ctx context.Context, rawURL string) (Preview, error) {
		return Preview{Title: "T"}, nil
	})

	const n = 10
	ids := make([]int64, n)
	for i := 0; i < n; i++ {
		idea, err := svc.Add(context.Background(), 1, "https://example.com/page")
		if err != nil {
			t.Fatalf("Add #%d: %v", i, err)
		}
		ids[i] = idea.ID
	}

	for _, id := range ids {
		resolved := waitForStatus(t, repo, id, 2*time.Second)
		if resolved.PreviewStatus != PreviewReady {
			t.Errorf("idea %d status = %q, want ready", id, resolved.PreviewStatus)
		}
	}
}
