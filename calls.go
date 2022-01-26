package main

import (
	// System
	"context"       // Context
	"encoding/json" // JSON
	"fmt"           // Formatting
	"io/ioutil"     // I/O
	"log"           // Logging
	"net/http"      // HTTP

	// Gcloud packages
	"cloud.google.com/go/storage" // Google Cloud Storage
	// Github packages
	"github.com/gorilla/mux" // Gorilla Mux
)

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

func createReportCall(w http.ResponseWriter, request *http.Request) {
	// Something something CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "multipart/form-data")
	// Log that we have received a request
	fmt.Println("Endpoint Hit: report/create")

	// Check the data is not larger than it should be
	if err := request.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get files from the request
	files := request.MultipartForm.File["file"]

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Loop through the files and upload them to the bucket
	for _, fileHeader := range files {
		fmt.Println(fileHeader.Filename)
		// Open the file
		file, _ := fileHeader.Open()
		// Read the file
		byteValue, _ := ioutil.ReadAll(file)

		// Create something to write the file to
		wc := client.Bucket(
			// Set the bucket
			"unwrapped-spotify-reports").Object(
			// Save in storageID/data/
			mux.Vars(request)["storageID"] +
				"/data/" +
				// Retain original filename
				fileHeader.Filename).NewWriter(ctx)

		// Pull content type from the fileheader
		wc.ContentType = fileHeader.Header["Content-Type"][0]

		// Try and write the file - log error if fails
		if _, err := wc.Write(byteValue); err != nil {
			log.Fatal(err)
			return
		}

		// Close the writer
		wc.Close()

	}

	// Build the report
	buildID := build(mux.Vars(request)["storageID"])

	// Respond with the build info - encode it as JSON
	json.NewEncoder(w).Encode(map[string]string{"buildID": buildID})

	//Return something here
}

func reportStatusCall(w http.ResponseWriter, request *http.Request) {
	// Something something CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	log.Println("Endpoint hit: Report status")
	buildID := request.FormValue("buildID")

	status := buildStatus(buildID)

	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func downloadReportCall(w http.ResponseWriter, request *http.Request) {
	// Something something CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/pdf")

	storageID := mux.Vars(request)["storageID"]

	ctx := context.Background()

	client, _ := storage.NewClient(ctx)

	file, _ := client.Bucket(
		"unwrapped-spotify-reports",
	).Object(
		storageID + "/output.pdf",
	).NewReader(ctx)

	fileBytes, _ := ioutil.ReadAll(file)

	w.Write(fileBytes)
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

func handleRequests() {
	// Create a router using the mux library
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/v1/healthcheck", healthcheckCall)
	router.HandleFunc("/api/v1/report/{storageID}/create", createReportCall).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/report/{storageID}/status", reportStatusCall).Queries("buildID", "{buildID}")
	router.HandleFunc("/api/v1/report/{storageID}/download.pdf", downloadReportCall)
	router.HandleFunc("/api/v1/users/{email}/create", createUserCall)
	//http.HandleFunc("/healthcheck", healthcheckCall)
	//http.HandleFunc("/streaming-history", streamingHistoryCall)
	//http.HandleFunc("/users/{email}/create", createUserCall)
	//log.Fatal(http.ListenAndServe(":500", nil))
	http.Handle("/", router)
	http.ListenAndServe(":500", nil)
}
