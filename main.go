package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"

	// "encoding/json"

	"cloud.google.com/go/storage"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	firestore "cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func healthcheckCall(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "{alive: true}")
	fmt.Println("Endpoint Hit: healthcheck")
}

func streamingHistoryCall(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	body, _ := ioutil.ReadAll(request.Body)
	bodyString := string(body)

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		// handle error.
	}
	wc := client.Bucket("unwrapped-spotify-reports").Object("data.json").NewWriter(ctx)
	wc.ContentType = "text/plain"

	if _, err := wc.Write([]byte(bodyString)); err != nil {
		fmt.Println("Unable to write data to bucket %v", err)
		return
	}

	wc.Close()

	build() // :)

	//Return something here
}

func handleRequests() {
	// Create a router using the mux library
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", healthcheckCall)
	router.HandleFunc("/streaming-history", streamingHistoryCall).Methods("POST", "OPTIONS")

	http.HandleFunc("/healthcheck", healthcheckCall)
	http.HandleFunc("/streaming-history", streamingHistoryCall)
	log.Fatal(http.ListenAndServe(":500", nil))
}

func createUser(email string) {
	projectID := "unwrapped-spotify"

	ctx := context.Background()

	client, err := firestore.NewClient(ctx, projectID)

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.Collection("users").Doc(email).Set(ctx, map[string]interface{}{
		"email":     email,
		"storageID": hash(email),
	})
}

func build() {
	ctx := context.Background()

	client, err := cloudbuild.NewClient(ctx)

	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	req := &cloudbuildpb.CreateBuildRequest{
		Parent:    "projects/unwrapped-spotify/locations/global",
		ProjectId: "unwrapped-spotify",
		Build: &cloudbuildpb.Build{
			//Source: &cloudbuildpb.Source{
			//	Source: &cloudbuildpb.Source_StorageSource{
			//		StorageSource: &cloudbuildpb.StorageSource{
			//			Bucket: "unwrapped-spotify-reports",
			//			Object: "data.json",
			//		},
			//	},
			//},
			Steps: []*cloudbuildpb.BuildStep{
				{
					Name: "gcr.io/cloud-builders/gsutil",
					Args: []string{"cp", "gs://unwrapped-spotify-reports/data.json", "data.json"},
				},
				{
					Name: "gcr.io/unwrapped-spotify/unwrapper",
				},
			},
			Artifacts: &cloudbuildpb.Artifacts{
				Objects: &cloudbuildpb.Artifacts_ArtifactObjects{
					Location: "gs://unwrapped-spotify-reports",
					Paths:    []string{"output.html"},
				},
			},
		},
	}

	response, err := client.CreateBuild(ctx, req)

	if err != nil {
		log.Fatal(err)
	}

	_ = response

}

func main() {
	fmt.Println("RESTful Go API starting on ")
	handleRequests()
}
