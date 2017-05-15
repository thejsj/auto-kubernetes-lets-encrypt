package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/xenolf/lego/acme"
)

type HealthResponse struct {
	Healthy bool
}

type ErrorResponse struct {
	Error         string
	originalError error
	Data          map[string]string
}

var CERTS_LOCATION = "/var/certs/"

func mainHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add validation for these
	domainsRaw := Getenv("DOMAINS", "")
	// TODO: Add email validation
	email := Getenv("EMAIL", "")
	// TODO: Make sure secret exists
	secretName := Getenv("SECRET_NAME", "")

	envInputs := make(map[string]string)
	envInputs["DOMAINS"] = domainsRaw
	envInputs["EMAIL"] = email
	envInputs["SECRET_NAME"] = secretName
	log.Printf("ENV inputs: %s", envInputs)
	if domainsRaw == "" || email == "" || secretName == "" {
		log.Printf("Environment variables not setup correctly: %s", envInputs)
		errorResponse := &ErrorResponse{
			Error: "The following ENV variables are required: `DOMAINS`, `EMAIL`, and `SECRET_NAME`",
			Data:  envInputs}
		SendError(w, errorResponse)
		return
	}
	// Get namespce
	log.Printf("Looking for kuberentes namespace in: %s", NAMESPACE_LOCATION)
	namespace, err := getNamespace()
	if err != nil {
		log.Printf("Kubernetes namespace not found in %s", NAMESPACE_LOCATION)
		namespaceInputs := make(map[string]string)
		namespaceInputs["NAMESPACE_LOCATION"] = NAMESPACE_LOCATION
		errorResponse := &ErrorResponse{
			Error: "Kubernetes namespace could not be found",
			Data:  namespaceInputs}
		SendError(w, errorResponse)
		return
	}
	log.Printf("Kubernetes namespace used: %s", namespace)
	log.Printf("Starting cert manager. Placing certs in: %s", CERTS_LOCATION)
	// Generate certiticates
	domains := strings.Split(domainsRaw, ",")
	for i := 0; i < len(domains); i++ {
		domains[i] = strings.Trim(domains[i], " ")
	}
	certErr := GenerateCerts(domains, email)
	log.Printf("Cert location", CERTS_LOCATION)
	log.Printf("Cert err: %s", certErr)
	// Update secret
	// Send response
	response := &HealthResponse{
		Healthy: true}
	SendJson(w, response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := &HealthResponse{
		Healthy: true}
	SendJson(w, response)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	email := Getenv("EMAIL", "")
	legoUser, err := getUser(email)
	if err != nil {
		errorResponse := &ErrorResponse{
			Error:         "Let's encrypt user not found",
			originalError: err}
		SendError(w, errorResponse)
		return
	}
	_, err = registerUser(legoUser)
	if err != nil {
		errorResponse := &ErrorResponse{
			Error:         "Could not register user",
			originalError: err}
		SendError(w, errorResponse)
		return
	}
	return
}

func GenerateCerts(domains []string, email string) error {
	legoUser, err := getUserWithRegistration(email)
	// https://github.com/xenolf/lego/blob/master/cli.go#L120
	caServerHost := Getenv("CA_SERVER", "https://acme-v01.api.letsencrypt.org/directory")
	log.Printf("Creating new user from CA server: %s", caServerHost)
	client, err := acme.NewClient(caServerHost, &legoUser, acme.RSA2048)
	if err != nil {
		log.Printf("Error creating acme client: %s", err)
		return err
	}
	// httpPort := Getenv("httpPort", "")
	// Let our server handle this, not lego
	client.SetHTTPAddress(":" + "5001")
	client.SetTLSAddress(":" + "5002")
	// New users will need to register
	err = client.AgreeToTOS()
	if err != nil {
		log.Printf("Error agreeing to terms of service: %s", err)
		return err
	}
	bundle := false
	certificates, failures := client.ObtainCertificate(domains, bundle, nil, false)
	log.Printf("%d failures founds", len(failures))
	fmt.Printf("%#v\n", certificates)
	if len(failures) > 0 {
		log.Printf("Too many failures: %s", failures)
		return err
	}
	return nil

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	fmt.Printf("%#v\n", certificates)
	return nil
}

func main() {
	http.HandleFunc("/", mainHandler)
	fs := http.FileServer(http.Dir("/.well-known"))
	http.Handle("/.well-known", fs)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/register", registrationHandler)
	httpPort := Getenv("httpPort", "80")
	log.Printf("HTTP Server listening on port: %s", httpPort)
	http.ListenAndServe(":"+httpPort, nil)
}
