package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/xenolf/lego/acme"
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
	fileData, err := ioutil.ReadFile(NAMESPACE_LOCATION)
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
	namespace := string(fileData)
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

func GenerateCerts(domains []string, email string) error {
	legoUser := getUser(email)
	client, err := acme.NewClient("http://192.168.99.100:4000", &legoUser, acme.RSA2048)
	if err != nil {
		log.Printf("Error creating acme client: %s", err)
		return err
	}
	httpPort := Getenv("httpPort", "80")
	client.SetHTTPAddress(":" + httpPort)
	client.SetTLSAddress(":" + httpPort)
	// New users will need to register
	reg, err := client.Register()
	if err != nil {
		log.Printf("Error registering user: %s", err)
		return nil
	}
	legoUser.Registration = reg
	log.Printf("User registered: %s", reg)
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

// You'll need a user or account type that implements acme.User
type LegoUser struct {
	Email        string
	Registration *acme.RegistrationResource
	key          crypto.PrivateKey
}

func (u LegoUser) GetEmail() string {
	return u.Email
}
func (u LegoUser) GetRegistration() *acme.RegistrationResource {
	return u.Registration
}
func (u LegoUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func getUser(email string) LegoUser {
	// Create a user. New accounts need an email and private key to start.
	const rsaKeySize = 2048
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		log.Fatal(err)
	}
	user := LegoUser{
		Email: email,
		key:   privateKey,
	}
	return user
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
	httpPort := Getenv("httpPort", "80")
	log.Printf("HTTP Server listening on port: %s", httpPort)
	http.ListenAndServe(":"+httpPort, nil)
}
