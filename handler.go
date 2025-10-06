package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/go-chi/chi/v5"
	"github.com/streadway/amqp"
	"googlesearchclone1.com/utils"
)

type Application struct {
	ES *elasticsearch.Client
	CH *amqp.Channel
}

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/complete/search", app.searchCompletionHandler)
	r.Get("/search", app.searchHandler)
	return r
}

func (app *Application) searchCompletionHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	var query = fmt.Sprintf(`{
            "suggest": {
                "term-suggester": {
                    "prefix": "%s",
                    "completion": {
                        "field": "suggest",
                        "size": 10
                    }
                }
            }
        }`, searchTerm)
	res, err := app.ES.Search(
		app.ES.Search.WithIndex("search-terms"),
		app.ES.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		log.Printf("elasticsearch search failed: %v", err)
		utils.Encode(w, r, http.StatusInternalServerError, err)
		return
	}
	defer res.Body.Close()

	var esResponse CompletionSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		log.Printf("failed to decode elasticsearch response: %v", err)
		utils.Encode(w, r, http.StatusInternalServerError, err)
		return
	}

	var suggestions []string
	for _, option := range esResponse.Suggest.TermSuggester[0].Options {
		suggestions = append(suggestions, option.Text)
	}

	err = app.CH.Publish(
		"",
		"search-terms",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(searchTerm),
		})
	if err != nil {
		log.Printf("failed to publish to rabbitmq: %v", err)
	}

	utils.Encode(w, r, http.StatusOK, suggestions)
}

func (app *Application) searchHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")

	if searchTerm == "" {
		utils.Encode(w, r, http.StatusInternalServerError, "missing search parameter")
		return
	}
	searchQuery := `{
        "query": {
            "multi_match": {
				"query": "%s",
				"fields": [ "title^3", "content" ],
				"fuzziness": "AUTO"
			}
        },
        "highlight": {
            "fields": {
                "content": {}
            }
        }
    }`

	requestBody := fmt.Sprintf(searchQuery, searchTerm)

	res, err := app.ES.Search(
		app.ES.Search.WithIndex("documents"),
		app.ES.Search.WithBody(strings.NewReader(requestBody)),
		app.ES.Search.WithTrackTotalHits(true),
	)
	defer res.Body.Close()

	var esResponse SearchResponse

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		log.Printf("failed to decode elasticsearch response: %v", err)
		utils.Encode(w, r, http.StatusInternalServerError, err)
		return
	}

	results := make([]SearchResult, 0, len(esResponse.Hits.Hits))
	for _, hit := range esResponse.Hits.Hits {

		result := SearchResult{
			Title:   hit.Source.Title,
			Content: hit.Source.Content,
		}

		results = append(results, result)
	}

	if err != nil {
		log.Printf("error retrieving documents %v", err)
		utils.Encode(w, r, http.StatusInternalServerError, err)
		return
	}
	utils.Encode(w, r, http.StatusOK, results)
}
