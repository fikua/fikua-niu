package httpapi

import (
	"time"

	"niu/internal/items"
	"niu/internal/projects"
)

// userDTO is the response shape for an embedded user reference
// (added_by/moved_by) or GET /api/v1/me.
type userDTO struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarEmoji string `json:"avatar_emoji"`
}

// itemDTO is the wire shape for an Item (design.md §6.1).
type itemDTO struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Location  string     `json:"location"`
	Position  float64    `json:"position"`
	AddedBy   *userDTO   `json:"added_by"`
	MovedBy   *userDTO   `json:"moved_by"`
	MovedAt   *time.Time `json:"moved_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func toUserDTO(u *items.User) *userDTO {
	if u == nil {
		return nil
	}
	return &userDTO{ID: u.ID, DisplayName: u.DisplayName, AvatarEmoji: u.AvatarEmoji}
}

func toItemDTO(it items.Item) itemDTO {
	return itemDTO{
		ID:        it.ID,
		Name:      it.Name,
		Location:  string(it.Location),
		Position:  it.Position,
		AddedBy:   toUserDTO(it.AddedBy),
		MovedBy:   toUserDTO(it.MovedBy),
		MovedAt:   it.MovedAt,
		CreatedAt: it.CreatedAt,
	}
}

func toItemDTOs(list []items.Item) []itemDTO {
	out := make([]itemDTO, 0, len(list))
	for _, it := range list {
		out = append(out, toItemDTO(it))
	}
	return out
}

// projectDTO is the wire shape for a Project (design.md §6.1). budget and
// target_date are null when not informed (AC-14/AC-15); last_updated_by
// is always non-nil, unlike itemDTO's moved_by (design.md §6.1 note: it
// is assigned equal to added_by at creation, so the frontend never needs
// the "has it ever moved" conditional that items.MovedBy requires).
type projectDTO struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	State         string    `json:"state"`
	Budget        *string   `json:"budget"`
	TargetDate    *string   `json:"target_date"`
	AddedBy       *userDTO  `json:"added_by"`
	LastUpdatedBy *userDTO  `json:"last_updated_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toProjectDTO(p projects.Project) projectDTO {
	return projectDTO{
		ID:            p.ID,
		Name:          p.Name,
		State:         string(p.State),
		Budget:        p.Budget,
		TargetDate:    p.TargetDate,
		AddedBy:       toUserDTO(p.AddedBy),
		LastUpdatedBy: toUserDTO(p.LastUpdatedBy),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func toProjectDTOs(list []projects.Project) []projectDTO {
	out := make([]projectDTO, 0, len(list))
	for _, p := range list {
		out = append(out, toProjectDTO(p))
	}
	return out
}
