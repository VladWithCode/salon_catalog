package db

import "time"

type Cart struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CartItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageURL  string `json:"image_url"`
	Quantity  int    `json:"quantity"`
	Source    string `json:"source"`               // "wizard" or "catalog"
	StepIndex int    `json:"step_index,omitempty"` // For wizard items
}
