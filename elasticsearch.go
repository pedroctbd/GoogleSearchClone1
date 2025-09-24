package main

type SuggestionDoc struct {
	Suggest struct {
		Input  []string `json:"input"`
		Weight int      `json:"weight"`
	} `json:"suggest"`
}

type SearchResponse struct {
	Suggest struct {
		TermSuggester []struct {
			Options []struct {
				Text   string        `json:"text"`
				Source SuggestionDoc `json:"_source"`
			} `json:"options"`
		} `json:"term-suggester"`
	} `json:"suggest"`
}
