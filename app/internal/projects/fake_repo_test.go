package projects

import (
	"context"
	"sync"
	"time"
)

// fakeRepo is an in-memory implementation of Repository and EventSink
// used for small/unit tests of Service — no SQLite involved, per the
// qa-engineer test pyramid (requirements.md §6).
//
// mu is load-bearing, not defensive boilerplate: since NIU-11 a preview
// scrape runs on a worker-pool goroutine (resolvePreview →
// UpdatePreview) while the test goroutine polls Get, so every map access
// here is genuinely concurrent. Without it `go test -race` fails
// deterministically. Mirrors internal/ideas/fake_repo_test.go, which has
// carried the same lock since NIU-6 for exactly this reason.
type fakeRepo struct {
	mu         sync.Mutex
	projects   map[int64]Project
	normalized map[int64]string // projectID -> name_normalized, mirrors the DB column
	events     []fakeEvent
	nextID     int64
	users      map[int64]User
}

type fakeEvent struct {
	userID  int64
	kind    string
	payload any
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:   make(map[int64]Project),
		normalized: make(map[int64]string),
		nextID:     1,
		users: map[int64]User{
			1: {ID: 1, Name: "usuari_a", DisplayName: "Usuari A", AvatarEmoji: "🐦"},
			2: {ID: 2, Name: "usuari_b", DisplayName: "Usuari B", AvatarEmoji: "🦊"},
		},
	}
}

func (f *fakeRepo) Create(ctx context.Context, userID int64, name, nameNormalized string, budget, targetDate, url *string) (Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.normalized {
		if n == nameNormalized {
			return Project{}, ErrDuplicate{}
		}
	}

	id := f.nextID
	f.nextID++
	u := f.users[userID]
	now := time.Now()

	var previewStatus *string
	if url != nil {
		v := PreviewPending
		previewStatus = &v
	}

	p := Project{
		ID:            id,
		Name:          name,
		State:         StateIdea,
		Budget:        budget,
		TargetDate:    targetDate,
		URL:           url,
		PreviewStatus: previewStatus,
		AddedBy:       &u,
		LastUpdatedBy: &u,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	f.projects[id] = p
	f.normalized[id] = nameNormalized
	return p, nil
}

// UpdatePreview mirrors ideas' fakeRepo.UpdatePreview (NIU-11): a
// deleted-while-scraping row is a silent no-op, never an error.
func (f *fakeRepo) UpdatePreview(ctx context.Context, id int64, title, imageURL, description *string, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.projects[id]
	if !ok {
		return nil
	}
	p.Title = title
	p.ImageURL = imageURL
	p.Description = description
	p.PreviewStatus = &status
	f.projects[id] = p
	return nil
}

func (f *fakeRepo) Get(ctx context.Context, id int64) (Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.projects[id]
	if !ok {
		return Project{}, ErrNotFound{ID: id}
	}
	return p, nil
}

func (f *fakeRepo) List(ctx context.Context) ([]Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []Project
	for _, p := range f.projects {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeRepo) UpdateState(ctx context.Context, id, userID int64, newState State) (Project, State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.projects[id]
	if !ok {
		return Project{}, "", ErrNotFound{ID: id}
	}
	previousState := p.State
	u := f.users[userID]
	p.State = newState
	p.LastUpdatedBy = &u
	p.UpdatedAt = time.Now()
	f.projects[id] = p
	return p, previousState, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.projects[id]; !ok {
		return false, nil
	}
	delete(f.projects, id)
	delete(f.normalized, id)
	return true, nil
}

func (f *fakeRepo) ExistsByNormalizedName(ctx context.Context, nameNormalized string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.normalized {
		if n == nameNormalized {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) Record(ctx context.Context, userID int64, kind string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, fakeEvent{userID: userID, kind: kind, payload: payload})
	return nil
}
