package main

type SuggestionDoc struct {
	Suggest struct {
		Input  []string `json:"input"`
		Weight int      `json:"weight"`
	} `json:"suggest"`
}

type CompletionSearchResponse struct {
	Suggest struct {
		TermSuggester []struct {
			Options []struct {
				Text   string        `json:"text"`
				Source SuggestionDoc `json:"_source"`
			} `json:"options"`
		} `json:"term-suggester"`
	} `json:"suggest"`
}

type SearchResponse struct {
	Hits struct {
		Hits []struct {
			Source struct {
				Title   string `json:"title"`
				Content string `json:"content"`
			} `json:"_source"`
			Highlight struct {
				Title   []string `json:"title"`
				Content []string `json:"content"`
			} `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
