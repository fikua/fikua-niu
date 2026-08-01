package items

import (
	"context"
	"time"
)

// fakeRepo is an in-memory implementation of Repository, EventSink and
// UserLookup used for small/unit tests of Service — no SQLite involved,
// per the qa-engineer test pyramid (requirements.md §6).
type fakeRepo struct {
	items      map[int64]Item
	normalized map[int64]string // itemID -> name_normalized, mirrors the DB column
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
		items:      make(map[int64]Item),
		normalized: make(map[int64]string),
		nextID:     1,
		users: map[int64]User{
			1: {ID: 1, Name: "usuari_a", DisplayName: "Usuari A", AvatarEmoji: "🐦"},
			2: {ID: 2, Name: "usuari_b", DisplayName: "Usuari B", AvatarEmoji: "🦊"},
		},
	}
}

func (f *fakeRepo) Create(ctx context.Context, userID int64, name, nameNormalized string) (Item, error) {
	for id, n := range f.normalized {
		if n == nameNormalized {
			return Item{}, ErrDuplicate{ExistingLocation: f.items[id].Location}
		}
	}

	maxPos, has, _ := f.MaxPosition(ctx, LocationShopping)
	pos := 1.0
	if has {
		pos = maxPos + 1.0
	}

	id := f.nextID
	f.nextID++
	u := f.users[userID]
	item := Item{
		ID:        id,
		Name:      name,
		Location:  LocationShopping,
		Position:  pos,
		AddedBy:   &u,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	f.items[id] = item
	f.normalized[id] = nameNormalized
	return item, nil
}

func (f *fakeRepo) Get(ctx context.Context, id int64) (Item, error) {
	it, ok := f.items[id]
	if !ok {
		return Item{}, ErrNotFound{ID: id}
	}
	return it, nil
}

func (f *fakeRepo) List(ctx context.Context) ([]Item, error) {
	var out []Item
	for _, it := range f.items {
		out = append(out, it)
	}
	return out, nil
}

func (f *fakeRepo) Update(ctx context.Context, id, userID int64, newLocation Location, position float64) (Item, error) {
	it, ok := f.items[id]
	if !ok {
		return Item{}, ErrNotFound{ID: id}
	}
	u := f.users[userID]
	now := time.Now()
	it.Location = newLocation
	it.MovedBy = &u
	it.MovedAt = &now
	it.UpdatedAt = now
	it.Position = position
	f.items[id] = it
	return it, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id int64) (bool, error) {
	if _, ok := f.items[id]; !ok {
		return false, nil
	}
	delete(f.items, id)
	delete(f.normalized, id)
	return true, nil
}

func (f *fakeRepo) ExistsByNormalizedName(ctx context.Context, nameNormalized string) (bool, Location, error) {
	for id, n := range f.normalized {
		if n == nameNormalized {
			return true, f.items[id].Location, nil
		}
	}
	return false, "", nil
}

func (f *fakeRepo) MaxPosition(ctx context.Context, location Location) (float64, bool, error) {
	max := 0.0
	has := false
	for _, it := range f.items {
		if it.Location == location {
			has = true
			if it.Position > max {
				max = it.Position
			}
		}
	}
	return max, has, nil
}

func (f *fakeRepo) Record(ctx context.Context, userID int64, kind string, payload any) error {
	f.events = append(f.events, fakeEvent{userID: userID, kind: kind, payload: payload})
	return nil
}

func (f *fakeRepo) GetUser(ctx context.Context, id int64) (User, error) {
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrNotFound{ID: id}
	}
	return u, nil
}
