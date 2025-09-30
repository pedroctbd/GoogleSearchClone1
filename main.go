package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or failed to load")
		log.Fatal()
	}

	//ElasticSearch
	esClient, err := SetupElasticSearchClient()
	if err != nil {
		log.Fatal(err)

	}
	//RabbitMQ config
	mqConn, mqChannel, err := SetupRabbitMqClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer mqConn.Close()
	defer mqChannel.Close()

	app := &Application{
		ES: esClient,
		MQ: mqChannel,
	}

	go StartWorker(esClient, mqChannel)

	http.ListenAndServe(":3000", app.routes())
}

func SetupElasticSearchClient() (*elasticsearch.Client, error) {

	cfg := elasticsearch.Config{

		Addresses: []string{
			"http://localhost:9200",
		},
		Username: os.Getenv("ELASTIC_USERNAME"),
		Password: os.Getenv("ELASTIC_PASSWORD"),
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Panicf("Error creating elastic client: %v", err)
	}
	_, err = client.Info()
	if err != nil {
		log.Panicf("Error showing elastic client info: %v", err)
	}

	indexName := "search-terms"
	mapping := `{
		"mappings": {
			"properties": {
				"suggest": {
					"type": "completion"
				}
			}
		}
	}`

	res, err := client.Indices.Exists([]string{indexName})
	if err != nil {
		log.Fatalf("Error checking if index exists: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {

		res, err := client.Indices.Create(
			indexName,
			client.Indices.Create.WithBody(strings.NewReader(mapping)),
		)

		fmt.Print(res)

		if err != nil {
			log.Fatalf("Error creating index: %s", err)
		}
		return client, nil
	}

	return client, nil

}
func SetupRabbitMqClient() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open a channel: %v", err)
		return nil, nil, err
	}

	q, err := ch.QueueDeclare(
		"search-terms",
		true,
		false,
		false,
		false,
		nil,
	)
	fmt.Print(q)
	if err != nil {
		log.Fatalf("failed to declare a queue: %v", err)
		return nil, nil, err
	}

	return conn, ch, nil
}
