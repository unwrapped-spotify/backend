package main

import (
	// System packages
	"context"       // Context
	"encoding/json" // Working with JSON
	"fmt"           // String formatting
	"log"           // Logging
	"os"            // Environment variables

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"                           // Cloud Build SDK
	firestore "cloud.google.com/go/firestore"                                   // Firestore
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1" // Cloud Build SDK
	"google.golang.org/protobuf/encoding/protojson"                             // JSON encoding
)

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

// Build the report
func build(storageID string) string {
	// Create a new context - should be done in the function definition
	ctx := context.Background()

	// Create a new client
	client, err := cloudbuild.NewClient(ctx)

	// Check for errors
	if err != nil {
		log.Fatal(err)
	}

	// Close when done
	defer client.Close()

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
					Args: []string{"cp", "-r", "gs://unwrapped-spotify-reports/" + storageID + "/data", "."},
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
					Paths:    []string{"output.pdf"},
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

	responseProto, _ := response.GetMetadata().UnmarshalNew()

	responseJson := protojson.Format(responseProto)
	var responseData map[string]interface{}

	json.Unmarshal([]byte(responseJson), &responseData)

	id := responseData["build"].(map[string]interface{})["id"]

	return (fmt.Sprintf("%v", id))

}

func buildStatus(buildID string) string {

	// Create a new context - should be done in the function definition
	ctx := context.Background()
	// Create a new client
	client, err := cloudbuild.NewClient(ctx)

	// Check for errors
	if err != nil {
		log.Fatal(err)
	}

	// Close when done
	defer client.Close()

	req := &cloudbuildpb.GetBuildRequest{
		ProjectId: os.Getenv("GCP_PROJECT_ID"),
		Id:        buildID,
	}

	build, _ := client.GetBuild(ctx, req)

	buildStatus := build.GetStatus().String()

	return (buildStatus)

}
