package main

import (
	"fmt"
	"log"
	"order"
)

func main() {
	err := order.Lint("config.yaml", "schema.json")
	if err != nil {
		log.Fatalf("error linting config.yaml file properties order against json schema: %v", err)
	}

	err = order.Lint("config.json", "schema.json")
	if err != nil {
		log.Fatalf("error linting config.json file properties order against json schema: %v", err)
	}

	fmt.Println("Properties order is valid")
}
