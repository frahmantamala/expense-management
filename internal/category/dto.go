package category

type CategoryResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
}
