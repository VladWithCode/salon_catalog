package db

type WizardCategory struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Required    bool            `json:"required"`
	MultiSelect bool            `json:"multi_select"`
	Order       int             `json:"order"`
	Products    []WizardProduct `json:"products"`
}

type WizardProduct struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
	Price       float64 `json:"price"`
	CategoryID  string  `json:"category_id"`
}
