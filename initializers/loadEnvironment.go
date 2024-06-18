package initializers

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnvironment() {
	err := godotenv.Load()
	if err != nil {
    log.Fatal("Error loading .env file")
	} else {
	log.Println("Loaded .env file")
	}
}