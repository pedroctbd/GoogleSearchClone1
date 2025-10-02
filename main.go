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
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Panicf("Error creating elastic client: %v", err)
	}
	_, err = es.Info()
	if err != nil {
		log.Panicf("Error showing elastic client info: %v", err)
	}

	suggestionsMapping := `{
		"mappings": {
			"properties": {
				"suggest": { "type": "completion" },
				"search_count": { "type": "integer" }
			}
		}
	}`

	documentsMapping := `{
		"mappings": {
			"properties": {
				"title": { "type": "text" },
				"url": { "type": "keyword" },
				"content": { "type": "text" }
			}
		}
	}`
	if err := createIndexIfNotExists(es, "search-terms", suggestionsMapping); err != nil {
		log.Fatalf("Error setting up search-terms index: %v", err)
	}

	if err := createIndexIfNotExists(es, "documents", documentsMapping); err != nil {
		log.Fatalf("Error setting up documents index: %v", err)
	}

	return es, nil

}

func createIndexIfNotExists(es *elasticsearch.Client, indexName string, mapping string) error {

	res, err := es.Indices.Exists([]string{indexName})
	if err != nil {
		return fmt.Errorf("cannot check if index exists: %w", err)
	}

	if res.StatusCode == http.StatusOK {
		return nil
	}

	if res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("error checking if index exists %s", err)
	}

	res, err = es.Indices.Create(indexName, es.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil {
		return fmt.Errorf("cannot create index: %w", err)
	}
	if res.IsError() {
		return fmt.Errorf("cannot create index: %s", res.String())
	}

	return nil
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
