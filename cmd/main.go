package main

import (
	"context"
	"log"
	"os"

	"github.com/djspoons/aqi2otel"
)

func main() {
	useStdoutExporter := false
	if len(os.Args) > 1 && os.Args[1] == "--stdout" {
		useStdoutExporter = true
	}
	log.Println("starting main()...")
	aqi2otel.Run(context.Background(), useStdoutExporter)
}
