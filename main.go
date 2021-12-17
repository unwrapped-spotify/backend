package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
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

func main() {
	fmt.Println("RESTful Go API starting on ")
	handleRequests()
}
