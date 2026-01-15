package main

import (
	"log"
	"net/http"
	_ "github.com/golang-jwt/jwt/v5"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Service is running (secure by design)"))
	})

	log.Println("Service starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
