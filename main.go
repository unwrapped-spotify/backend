package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	"github.com/gorilla/mux"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

func healthcheckCall(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "{alive: true}")
	fmt.Println("Endpoint Hit: healthcheck")
}

func handleRequests() {
	// Create a router using the mux library
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", healthcheckCall)

	http.HandleFunc("/healthcheck", healthcheckCall)
	log.Fatal(http.ListenAndServe(":500", nil))
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
			Source: &cloudbuildpb.Source{
				Source: &cloudbuildpb.Source_StorageSource{
					StorageSource: &cloudbuildpb.StorageSource{
						Bucket: "unwrapped-spotify-reports",
						Object: "data.zip",
					},
				},
			},
			Steps: []*cloudbuildpb.BuildStep{
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
	//handleRequests()
	build()
}
