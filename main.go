package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"googlesearchclone1.com/utils"
)

func main() {

	// ctx := context.Background()
	r := chi.NewRouter()

	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or failed to load")
	}

	es, _ := elasticsearch.NewDefaultClient()
	log.Println(es.Info())

	r.Get("/complete/search", func(w http.ResponseWriter, r *http.Request) {

		searchPrefix := r.URL.Query().Get("q")
		log.Println(searchPrefix)
		utils.Encode(w, r, http.StatusAccepted, fmt.Sprintf("current prefix = %s", searchPrefix))
		return

	})

	http.ListenAndServe(":3000", r)
}
