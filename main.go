package main

// Import packages
import (
	// System packages
	"errors"
	"fmt"      // IO pipes
	"hash/fnv" // Used to hash the email
	"log"      // Logging

	// Working with HTTP requests
	"os" // Environment variables

	// Github packages
	"github.com/joho/godotenv" // Used to load environment variables
)

// This hashes a string into a number
func hash(s string) uint32 {
	// Create a new hash
	h := fnv.New32a()
	// Write the string to the hash
	h.Write([]byte(s))
	// Return the hash
	return h.Sum32()
}

func main() {
	// Load the environment variables
	if _, err := os.Stat(".env"); errors.Is(err, os.ErrNotExist) {
		log.Print("No .env file found, skipping")
	} else {
		err := godotenv.Load()
		// Check for errors
		if err != nil {
			log.Fatal("Error loading .env file")
			log.Fatal(err)
		}
	}
	// Print that we are running
	fmt.Println("RESTful Go API starting on ")
	// Now start running...
	handleRequests()
}
