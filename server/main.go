package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/acme/autocert"
)

type HealthResponse struct {
	Healthy bool
}

type ErrorResponse struct {
	Error string
	Data  map[string]string
}

var NAMESPACE_LOCATION = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
var CERTS_LOCATION = "/var/certs/"

func mainHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add validation for these
	domains := Getenv("DOMAINS", "")
	// TODO: Add email validation
	email := Getenv("EMAIL", "")
	// TODO: Make sure secret exists
	secretName := Getenv("SECRET_NAME", "")

	envInputs := make(map[string]string)
	envInputs["DOMAINS"] = domains
	envInputs["EMAIL"] = email
	envInputs["SECRET_NAME"] = secretName
	log.Printf("ENV inputs: %s", envInputs)
	if domains == "" || email == "" || secretName == "" {
		log.Error("Environment variables not setup correctly: %s", envInputs)
		errorResponse := &ErrorResponse{
			Error: "The following ENV variables are required: `DOMAINS`, `EMAIL`, and `SECRET_NAME`",
			Data:  envInputs}
		SendError(w, errorResponse)
		return
	}
	// Get namespce
	log.Printf("Looking for kuberentes namespace in: %s", NAMESPACE_LOCATION)
	fileData, err := ioutil.ReadFile(NAMESPACE_LOCATION)
	if err != nil {
		log.Error("Kubernetes namespace not found in %s", NAMESPACE_LOCATION)
		namespaceInputs := make(map[string]string)
		namespaceInputs["NAMESPACE_LOCATION"] = NAMESPACE_LOCATION
		errorResponse := &ErrorResponse{
			Error: "Kubernetes namespace could not be found",
			Data:  namespaceInputs}
		SendError(w, errorResponse)
		return
	}
	namespace := string(fileData)
	log.Printf("Kubernetes namespace used: %s", namespace)
	log.Printf("Starting cert manager. Placing certs in: %s", CERTS_LOCATION)
	// Generate certiticates
	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		// TODO: Handle multiple domains
		HostPolicy: autocert.HostWhitelist(domains),
		Cache:      autocert.DirCache(CERTS_LOCATION),
		Email:      email,
	}
	hello := &tls.ClientHelloInfo{
		ServerName: domains,
	}
	certManager.GetCertificate(hello)
	log.Printf("Cert location", CERTS_LOCATION)

	// Update secret

	// Send response
	SendJson(w, response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := &HealthResponse{
		Healthy: true}
	SendJson(w, response)
}

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

func main() {
	http.HandleFunc("/", mainHandler)
	fs := http.FileServer(http.Dir("/.well-known"))
	http.Handle("/.well-known", fs)
	http.HandleFunc("/health", healthHandler)
	httpPort := Getenv("httpPort", "8000")
	log.Printf("HTTP Server listening on port: %s", httpPort)
	http.ListenAndServe(":"httpPort, nil)
	http.ListenAndServe(":443"+, nil)
}
