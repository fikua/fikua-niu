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

// projectDTO mirrors the wire shape of httpapi.projectDTO (design.md
// §6.1) — duplicated here because the field is unexported in
// internal/httpapi; integration tests only care about the JSON contract.
// NIU-11: url/title/image_url/preview_status are the exposed preview
// fields; description is deliberately absent — see
// TestProjects_Add_WithURL_DescriptionNeverExposed, which asserts its
// absence from the raw JSON body directly rather than relying on this
// struct silently dropping an unknown field.
type projectDTO struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	State         string    `json:"state"`
	Budget        *string   `json:"budget"`
	TargetDate    *string   `json:"target_date"`
	URL           *string   `json:"url"`
	Title         *string   `json:"title"`
	ImageURL      *string   `json:"image_url"`
	PreviewStatus *string   `json:"preview_status"`
	AddedBy       *userDTO  `json:"added_by"`
	LastUpdatedBy *userDTO  `json:"last_updated_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type projectsListResponse struct {
	Projects []projectDTO `json:"projects"`
}

// ideaDTO mirrors the wire shape of httpapi.ideaDTO (design.md §6.1) —
// duplicated here because the field is unexported in internal/httpapi;
// integration tests only care about the JSON contract.
type ideaDTO struct {
	ID            int64     `json:"id"`
	URL           string    `json:"url"`
	Title         *string   `json:"title"`
	ImageURL      *string   `json:"image_url"`
	Description   *string   `json:"description"`
	PreviewStatus string    `json:"preview_status"`
	AddedBy       *userDTO  `json:"added_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type ideasListResponse struct {
	Ideas []ideaDTO `json:"ideas"`
}
