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
	MQ *amqp.Channel
}

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/complete/search", app.searchHandler)
	return r
}

func (app *Application) searchHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("ERROR: elasticsearch search failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	var esResponse SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		log.Printf("ERROR: failed to decode elasticsearch response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var suggestions []string
	for _, option := range esResponse.Suggest.TermSuggester[0].Options {
		suggestions = append(suggestions, option.Text)
	}

	err = app.MQ.Publish(
		"",
		"search-terms",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(searchTerm),
		})
	if err != nil {
		// Just log this error, don't fail the user's request
		log.Printf("WARNING: failed to publish to rabbitmq: %v", err)
	}

	// --- Respond to Client ---
	utils.Encode(w, r, http.StatusOK, suggestions)
}
