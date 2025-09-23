package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"googlesearchclone1.com/utils"
)

type Document struct {
	Prefix    string    `json:"prefix"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type QueueTask struct {
	Id        string
	Prefix    string
	updatedAt time.Time
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

	//RabbitMQ config
	conn, err := amqp.Dial(os.Getenv("http://localhost:5672"))
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"jobs",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare a queue: %v", err)
	}

	r.Get("/complete/search", func(w http.ResponseWriter, r *http.Request) {

		searchPrefix := r.URL.Query().Get("q")

		//update count
		newM := QueueTask{
			Id:        "1",
			updatedAt: time.Now(),
			Prefix:    searchPrefix,
		}
		body, err := json.Marshal(newM)
		if err != nil {
			log.Fatalf("failed to marshal message: %v", err)
		}
		err = ch.Publish(
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        body,
			},
		)
		if err != nil {
			log.Fatalf("failed to publish a message: %v", err)
		}

		utils.Encode(w, r, http.StatusAccepted, fmt.Sprintf("current prefix = %s", searchPrefix))
		return

	})

	http.ListenAndServe(":3000", r)
}
