package category

// CategoryResponse represents a category in API responses
type CategoryResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CategoriesResponse represents a collection of categories in API responses
type CategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
}
