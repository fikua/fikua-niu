package httpapi

import (
	"time"

	"niu/internal/items"
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
