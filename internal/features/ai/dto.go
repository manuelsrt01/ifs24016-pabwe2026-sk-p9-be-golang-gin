package ai

type GenerateBlogRequest struct {
	Topic            string `json:"topic"`
	Category         string `json:"category"`
	IncludeWebSearch *bool  `json:"include_web_search,omitempty"`
}

type SearchSource struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type GeneratedBlogResponse struct {
	Title         string         `json:"title"`
	Category      string         `json:"category"`
	Excerpt       string         `json:"excerpt"`
	Content       string         `json:"content"`
	Sources       []SearchSource `json:"sources"`
	SearchSources []SearchSource `json:"search_sources"`
}
