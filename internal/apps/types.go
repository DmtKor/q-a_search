package apps

import "time"

// App is the API/DB model (OpenAPI App).
type App struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Settings  map[string]interface{} `json:"settings"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// AppCreate is the create body (OpenAPI AppCreate).
type AppCreate struct {
	Name     string                 `json:"name"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// AppUpdate is the update body (OpenAPI AppUpdate); all fields optional.
type AppUpdate struct {
	Name     *string                `json:"name,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}
