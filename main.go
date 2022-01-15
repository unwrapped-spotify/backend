package main

// Import packages
import (
	// System packages
	"context" // ICL no clue what this is
	"encoding/json"
	"errors"
	"fmt"      // IO pipes
	"hash/fnv" // Used to hash the email
	"io/ioutil"
	"log"      // Logging
	"net/http" // Working with HTTP requests
	"os"       // Environment variables

	// Github packages
	"github.com/gorilla/mux"   // Mux provides URL routing
	"github.com/joho/godotenv" // Used to load environment variables

	// Gcloud packages
	"cloud.google.com/go/storage" // Cloud Storage (buckets)

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"                           // Cloud Build
	firestore "cloud.google.com/go/firestore"                                   // Cloud Firestore
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1" // Extra bits for cloudbuild
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

// Provides a healthcheck endpoint to see if API is running
func healthcheckCall(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Create a JSON to return to client to signal we are alive
	json.NewEncoder(w).Encode(map[string]bool{"alive": true})
	// Log that we have received a request
	fmt.Println("Endpoint Hit: healthcheck")
}

func streamingHistoryCall(w http.ResponseWriter, request *http.Request) {
	// Something something CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Get the JSON from the body of the request
	body, _ := ioutil.ReadAll(request.Body)
	bodyString := string(body)

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	wc := client.Bucket("unwrapped-spotify-reports").Object("/" + mux.Vars(request)["storageID"] + "/data.json").NewWriter(ctx)
	wc.ContentType = "text/plain"

	if _, err := wc.Write([]byte(bodyString)); err != nil {
		log.Fatal(err)
		return
	}

	wc.Close()

	build(mux.Vars(request)["storageID"]) // :)

	//Return something here
}

func createUserCall(w http.ResponseWriter, request *http.Request) {
	// Something something CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	storageID := createUser(mux.Vars(request)["email"])
	// Respond with the storage ID - encode it as JSON
	json.NewEncoder(w).Encode(map[string]uint32{"storageID": storageID})
}

// Create a user in the firestore database - returns the storage ID/hashed email
func createUser(email string) uint32 {
	// Create a new context - should be done in the function definition
	ctx := context.Background()
	// Create a new client
	client, err := firestore.NewClient(ctx, os.Getenv("GCP_PROJECT_ID"))

	// Check for errors
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create a new document for the user. The path will be users/email
	client.Collection("users").Doc(email).Set(ctx, map[string]interface{}{
		// Save the email
		"email": email,
		// Create a storage ID, this is a hash of the email string so is unique for each document/user
		"storageID": hash(email),
	})

	return hash(email)
}

func handleRequests() {
	// Create a router using the mux library
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/v1/healthcheck", healthcheckCall)
	router.HandleFunc("/api/v1/streaming-history/{storageID}/build", streamingHistoryCall).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/users/{email}/create", createUserCall)
	//http.HandleFunc("/healthcheck", healthcheckCall)
	//http.HandleFunc("/streaming-history", streamingHistoryCall)
	//http.HandleFunc("/users/{email}/create", createUserCall)
	//log.Fatal(http.ListenAndServe(":500", nil))
	http.Handle("/", router)
	http.ListenAndServe(":500", nil)
}

// Build the report
func build(storageID string) {
	// Create a new context - should be done in the function definition
	ctx := context.Background()

	// Create a new client
	client, err := cloudbuild.NewClient(ctx)
	// Close when done
	defer client.Close()

	// Check for errors
	if err != nil {
		log.Fatal(err)
	}

	// Create a new build request - this will build the report by running the R container
	req := &cloudbuildpb.CreateBuildRequest{
		// Where to create the resource
		Parent: "projects/" + os.Getenv("GCP_PROJECT_ID") + "/locations/global",
		// Porject ID - this is the project that the container will be run in
		ProjectId: os.Getenv("GCP_PROJECT_ID"),
		// The build to run
		Build: &cloudbuildpb.Build{
			// Build constists of 2 steps.
			// 1. Copy the data.json from the bucket to the workspace
			// 2. Run the R container/report builder
			Steps: []*cloudbuildpb.BuildStep{
				// Step 1
				{
					Name: "gcr.io/cloud-builders/gsutil",
					Args: []string{"cp", "gs://unwrapped-spotify-reports/" + storageID + "/data.json", "data.json"},
				},
				// Step 2
				{
					Name: "gcr.io/unwrapped-spotify/unwrapper",
				},
			},
			// The report is an artifact - copy this to the bucket
			Artifacts: &cloudbuildpb.Artifacts{
				Objects: &cloudbuildpb.Artifacts_ArtifactObjects{
					Location: "gs://unwrapped-spotify-reports/" + storageID,
					Paths:    []string{"output.html"},
				},
			},
		},
	}

	// Start the build
	response, err := client.CreateBuild(ctx, req)

	// Check for errors
	if err != nil {
		log.Fatal(err)
	}

	// Bin off response for now
	_ = response

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
