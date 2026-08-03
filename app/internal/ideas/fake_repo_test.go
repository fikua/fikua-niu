package ideas

import (
	"context"
	"sync"
	"time"
)

// fakeRepo is an in-memory implementation of Repository and EventSink
// used for small/unit tests of Service — no SQLite involved, per the
// qa-engineer test pyramid (requirements.md §6).
type fakeRepo struct {
	mu     sync.Mutex
	ideas  map[int64]Idea
	events []fakeEvent
	nextID int64
	users  map[int64]User
}

type fakeEvent struct {
	userID  int64
	kind    string
	payload any
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		ideas:  make(map[int64]Idea),
		nextID: 1,
		users: map[int64]User{
			1: {ID: 1, Name: "usuari_a", DisplayName: "Usuari A", AvatarEmoji: "🐦"},
			2: {ID: 2, Name: "usuari_b", DisplayName: "Usuari B", AvatarEmoji: "🦊"},
		},
	}
}

func (f *fakeRepo) Create(ctx context.Context, userID int64, url string) (Idea, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.nextID
	f.nextID++
	u := f.users[userID]
	idea := Idea{
		ID:            id,
		URL:           url,
		PreviewStatus: PreviewPending,
		AddedBy:       &u,
		CreatedAt:     time.Now(),
	}
	f.ideas[id] = idea
	return idea, nil
}

func (f *fakeRepo) Get(ctx context.Context, id int64) (Idea, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	idea, ok := f.ideas[id]
	if !ok {
		return Idea{}, ErrNotFound{ID: id}
	}
	return idea, nil
}

func (f *fakeRepo) List(ctx context.Context) ([]Idea, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []Idea
	for _, idea := range f.ideas {
		out = append(out, idea)
	}
	return out, nil
}

func (f *fakeRepo) UpdatePreview(ctx context.Context, id int64, title, imageURL, description *string, status PreviewStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	idea, ok := f.ideas[id]
	if !ok {
		// Mirrors the real repository: a deleted-while-scraping row is a
		// silent no-op, never an error (design.md §5 Flux 3).
		return nil
	}
	idea.Title = title
	idea.ImageURL = imageURL
	idea.Description = description
	idea.PreviewStatus = status
	f.ideas[id] = idea
	return nil
}

func (f *fakeRepo) Delete(ctx context.Context, id int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.ideas[id]; !ok {
		return false, nil
	}
	delete(f.ideas, id)
	return true, nil
}

func (f *fakeRepo) Record(ctx context.Context, userID int64, kind string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, fakeEvent{userID: userID, kind: kind, payload: payload})
	return nil
}
