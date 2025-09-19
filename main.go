package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"googlesearchclone1.com/utils"
)

type Document struct {
	Prefix    string    `json:"prefix"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {

	r := chi.NewRouter()

	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or failed to load")
	}

	cfg := elasticsearch.Config{

		Addresses: []string{
			"http://localhost:9200",
		},
		Username: os.Getenv("ELASTIC_USERNAME"),
		Password: os.Getenv("ELASTIC_PASSWORD"),
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	_, err = client.Info()
	if err != nil {
		panic(err)
	}

	r.Get("/complete/search", func(w http.ResponseWriter, r *http.Request) {

		searchPrefix := r.URL.Query().Get("q")

		utils.Encode(w, r, http.StatusAccepted, fmt.Sprintf("current prefix = %s", searchPrefix))
		return

	})

	http.ListenAndServe(":3000", r)
}
