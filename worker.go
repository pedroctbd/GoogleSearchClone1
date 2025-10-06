package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/streadway/amqp"
)

func StartWorker(es *elasticsearch.Client, ch *amqp.Channel) {
	msgs, err := ch.Consume(
		"search-terms",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to register a consumer: %v", err)
	}

	log.Printf("Worker is waiting for messages.")

	for d := range msgs {
		searchTerm := string(d.Body)
		log.Printf("Received a message: %s", searchTerm)
		err := updateSuggestion(es, searchTerm)
		if err != nil {
			log.Printf("error processing message %s", err)
			d.Nack(false, true)
		} else {
			d.Ack(false)
		}
	}

}

func updateSuggestion(es *elasticsearch.Client, searchTerm string) error {

	docID := searchTerm

	escapedTerm, err := json.Marshal(searchTerm)
	if err != nil {
		return fmt.Errorf("failed to marshal search term: %w", err)
	}

	scriptTemplate := `
	{
        "scripted_upsert": true,
        "script": {
            "source": "ctx._source.search_count += 1; ctx._source.suggest.weight = ctx._source.search_count;",
            "lang": "painless"
        },
        "upsert": {
            "suggest": {
                "input": [%s],
                "weight": 1
            },
            "search_count": 1
        }
    }`

	requestBody := fmt.Sprintf(scriptTemplate, escapedTerm)

	res, err := es.Update(
		"search-terms",
		docID,
		strings.NewReader(requestBody),
		es.Update.WithContext(context.Background()),
		es.Update.WithRefresh("true"),
	)
	if err != nil {
		log.Printf("Error updating document %s: %s", docID, err)
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error response from Elasticsearch for doc %s: %s", docID, res.String())
		log.Print(err)
		return err
	}

	log.Printf("Successfully updated document: %s", docID)
	return nil
}
