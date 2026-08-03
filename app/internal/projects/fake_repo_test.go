package projects

import (
	"context"
	"time"
)

// fakeRepo is an in-memory implementation of Repository and EventSink
// used for small/unit tests of Service — no SQLite involved, per the
// qa-engineer test pyramid (requirements.md §6).
type fakeRepo struct {
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

func (f *fakeRepo) Create(ctx context.Context, userID int64, name, nameNormalized string, budget, targetDate *string) (Project, error) {
	for _, n := range f.normalized {
		if n == nameNormalized {
			return Project{}, ErrDuplicate{}
		}
	}

	id := f.nextID
	f.nextID++
	u := f.users[userID]
	now := time.Now()
	p := Project{
		ID:            id,
		Name:          name,
		State:         StateIdea,
		Budget:        budget,
		TargetDate:    targetDate,
		AddedBy:       &u,
		LastUpdatedBy: &u,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	f.projects[id] = p
	f.normalized[id] = nameNormalized
	return p, nil
}

func (f *fakeRepo) Get(ctx context.Context, id int64) (Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return Project{}, ErrNotFound{ID: id}
	}
	return p, nil
}

func (f *fakeRepo) List(ctx context.Context) ([]Project, error) {
	var out []Project
	for _, p := range f.projects {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeRepo) UpdateState(ctx context.Context, id, userID int64, newState State) (Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return Project{}, ErrNotFound{ID: id}
	}
	u := f.users[userID]
	p.State = newState
	p.LastUpdatedBy = &u
	p.UpdatedAt = time.Now()
	f.projects[id] = p
	return p, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id int64) (bool, error) {
	if _, ok := f.projects[id]; !ok {
		return false, nil
	}
	delete(f.projects, id)
	delete(f.normalized, id)
	return true, nil
}

func (f *fakeRepo) ExistsByNormalizedName(ctx context.Context, nameNormalized string) (bool, error) {
	for _, n := range f.normalized {
		if n == nameNormalized {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) Record(ctx context.Context, userID int64, kind string, payload any) error {
	f.events = append(f.events, fakeEvent{userID: userID, kind: kind, payload: payload})
	return nil
}
