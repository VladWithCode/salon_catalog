package db

type CartItem struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageURL  string `json:"image_url"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
	Source    string `json:"source"`               // "wizard" or "catalog"
	StepIndex int    `json:"step_index,omitempty"` // For wizard items
}
