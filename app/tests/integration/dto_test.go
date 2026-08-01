package integration

import "time"

// itemDTO mirrors the wire shape of httpapi.itemDTO (design.md §6.1) —
// duplicated here because the field is unexported in internal/httpapi;
// integration tests only care about the JSON contract, not the Go type.
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

type userDTO struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarEmoji string `json:"avatar_emoji"`
}

type itemsListResponse struct {
	Items []itemDTO `json:"items"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
