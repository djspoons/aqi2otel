package aqi2otel

import (
	"context"
	"log"
)

// PubSubMessage is the payload of a Pub/Sub event. Please refer to the docs for
// additional information regarding Pub/Sub events.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// Entry is the entry point when run as a Cloud Function
func Entry(ctx context.Context, m PubSubMessage) error {
	log.Println("starting Entry() with message: " + string(m.Data))
	Run(ctx, false)
	return nil
}
