package main

import (
	"fmt"
	"log"
	"net/http"
)

func healthcheckCall(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "{success: true}")
	fmt.Println("Endpoint Hit: healthcheck")
}

func handleRequests() {
	http.HandleFunc("/healthcheck", healthcheckCall)
	log.Fatal(http.ListenAndServe(":500", nil))
}

func main() {
	handleRequests()
}
