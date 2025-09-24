package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ptrciafae/hotels-merge/internal/hotels"
	"github.com/ptrciafae/hotels-merge/internal/mapper"
	"github.com/ptrciafae/hotels-merge/internal/server"
)

func main() {

	store := hotels.NewHotelStore()

	// load mapping configuration from file
	file, err := os.Open("./mapping.json")
	if err != nil {
		fmt.Printf("error opening mapping file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	mappingConfig, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("error reading mapping file: %v\n", err)
		os.Exit(1)
	}

	engine, err := mapper.NewMappingEngine(mappingConfig)
	if err != nil {
		fmt.Printf("error creating mapping engine: %v\n", err)
		os.Exit(1)
	}

	hotels, err := hotels.FetchAndNormalize(engine)
	if err != nil {
		fmt.Printf("error fetching and normalizing hotels: %v\n", err)
		os.Exit(1)
	}
	store.Set(hotels)
	srv := server.New(store)

	log.Println("Server starting on :8085")
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
