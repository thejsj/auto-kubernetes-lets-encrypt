package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func SendError(w http.ResponseWriter, response interface{}) {
	w.WriteHeader(http.StatusBadRequest)
	SendJson(w, response)
}

func SendJson(w http.ResponseWriter, response interface{}) {
	json, err := json.Marshal(response)
	if err != nil {
		log.Fatal("Error marshalling HTTP response: %s - %s", response, err)
		panic(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
