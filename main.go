package main

import (
	"log"
	"net/http"
)

func main() {
	handler := NewServer()

	log.Println("Server is listen on :8080 ")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

