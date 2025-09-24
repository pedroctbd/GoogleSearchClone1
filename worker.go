package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/streadway/amqp"
)

// StartWorker starts the RabbitMQ consumer
func StartWorker(es *elasticsearch.Client, ch *amqp.Channel) {
	msgs, err := ch.Consume(
		"search-terms", // Use the same queue name as your main app
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		log.Fatalf("failed to register a consumer: %v", err)
	}

	log.Printf(" [*] Worker is waiting for messages.")

	go func() {
		for d := range msgs {
			searchTerm := string(d.Body)
			log.Printf("Received a message: %s", searchTerm)
			updateSuggestion(es, searchTerm)
		}
	}()
}

func updateSuggestion(es *elasticsearch.Client, searchTerm string) {
	docID := searchTerm

	script := `{
		"script": {
			"source": "ctx._source.suggest.weight += 1",
		},
		"upsert": {
			"suggest": {
				"input": ["%s"],
				"weight": 1
			}
		}
	}`

	scriptWithTerm := fmt.Sprintf(script, searchTerm)
	_, err := es.Update("search-terms", docID, strings.NewReader(scriptWithTerm), es.Update.WithContext(context.Background()))

	if err != nil {
		log.Printf("Error updating document %s: %s", docID, err)
	} else {
		log.Printf("Successfully updated document: %s", docID)
	}
}
