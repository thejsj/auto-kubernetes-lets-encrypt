package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/xenolf/lego/acme"
	"github.com/xenolf/lego/providers/http/webroot"
)

type HealthResponse struct {
	Healthy bool
}

type SuccessResponse struct {
	Success bool
	Message string
}

type ErrorResponse struct {
	Error         string
	originalError error
	Data          map[string]string
}

var CERTS_LOCATION = "/var/certs/"
var WEBROOT_LOCATION = "/var/www/"
var IN_PROGRESS = false

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if IN_PROGRESS {
		log.Printf("Refusing to start process because process is currently in progress")
		errorResponse := &ErrorResponse{
			Error: "Process is currently in progress"}
		SendError(w, errorResponse)
		return
	}
	IN_PROGRESS = true

	log.Printf("Start main handler...")
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
	log.Printf("Cert location", CERTS_LOCATION)
	certErr := GenerateCerts(domains, email)
	if certErr != nil {
		log.Printf("Cert err: %s", certErr)
		errorResponse := &ErrorResponse{
			Error:         "Cannot get user registration. User has not be registered or registration cannot be properly retrieved.",
			originalError: certErr}
		SendError(w, errorResponse)
		return
	}
	// Update secret
	// Send response
	response := &SuccessResponse{
		Success: true,
		Message: "Your certs have been successfully created"}
	IN_PROGRESS = false
	SendJson(w, response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Healthness Check")
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
	message := fmt.Sprintf("User successfully created: %s", &legoUser.Registration)
	response := &SuccessResponse{
		Success: true,
		Message: message}
	SendJson(w, response)
	return
}

func GenerateCerts(domains []string, email string) error {
	legoUser, err := getUserWithRegistration(email)
	if err != nil {
		log.Printf("Error getting user with registration: %s", err)
		return err
	}
	// https://github.com/xenolf/lego/blob/master/cli.go#L120
	caServerHost := Getenv("CA_SERVER", "https://acme-v01.api.letsencrypt.org/directory")
	log.Printf("Creating new user from CA server: %s", caServerHost)
	client, err := acme.NewClient(caServerHost, &legoUser, acme.RSA2048)
	if err != nil {
		log.Printf("Error creating acme client: %s", err)
		return err
	}

	log.Printf("Setting webroot provider at %s", WEBROOT_LOCATION)
	provider, err := webroot.NewHTTPProvider(WEBROOT_LOCATION)
	if err != nil {
		log.Printf("Error creating acme client provider: %s", err)
		return err
	}
	log.Printf("Setting challenge provider to HTTP")
	client.SetChallengeProvider(acme.HTTP01, provider)
	log.Printf("Excluding all other challenges")
	client.ExcludeChallenges([]acme.Challenge{acme.DNS01, acme.TLSSNI01})

	// New users will need to register
	log.Printf("Agreeing to TOS")
	err = client.AgreeToTOS()
	if err != nil {
		log.Printf("Error agreeing to terms of service: %s", err)
		return err
	}
	bundle := false
	log.Printf("Obtaining certificates...")
	certificates, failures := client.ObtainCertificate(domains, bundle, nil, false)
	log.Printf("%d failures founds", len(failures))
	if len(failures) > 0 {
		log.Printf("Too many failures: %s", failures)
		return err
	}

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	fmt.Printf("%#v\n", certificates)
	log.Printf("Save certs to disk")
	saveCertToDisk(certificates, "/etc/auto-kubernetes-lets-encrypt/certs/")
	return nil
}

func saveCertToDisk(certRes acme.CertificateResource, certPath string) {
	// We store the certificate, private key and metadata in different files
	// as web servers would not be able to work with a combined file.
	certOut := path.Join(certPath, certRes.Domain+".crt")
	privOut := path.Join(certPath, certRes.Domain+".key")
	pemOut := path.Join(certPath, certRes.Domain+".pem")
	metaOut := path.Join(certPath, certRes.Domain+".json")
	issuerOut := path.Join(certPath, certRes.Domain+".issuer.crt")

	err := ioutil.WriteFile(certOut, certRes.Certificate, 0600)
	if err != nil {
		log.Fatalf("Unable to save Certificate for domain %s\n\t%s", certRes.Domain, err.Error())
	}

	if certRes.IssuerCertificate != nil {
		err = ioutil.WriteFile(issuerOut, certRes.IssuerCertificate, 0600)
		if err != nil {
			log.Fatalf("Unable to save IssuerCertificate for domain %s\n\t%s", certRes.Domain, err.Error())
		}
	}

	if certRes.PrivateKey != nil {
		// if we were given a CSR, we don't know the private key
		err = ioutil.WriteFile(privOut, certRes.PrivateKey, 0600)
		if err != nil {
			log.Fatalf("Unable to save PrivateKey for domain %s\n\t%s", certRes.Domain, err.Error())
		}

		err = ioutil.WriteFile(pemOut, bytes.Join([][]byte{certRes.Certificate, certRes.PrivateKey}, nil), 0600)
		if err != nil {
			log.Fatalf("Unable to save Certificate and PrivateKey in .pem for domain %s\n\t%s", certRes.Domain, err.Error())
		}
	} else {
		// we don't have the private key; can't write the .pem file
		log.Fatalf("Unable to save pem without private key for domain %s\n\t%s; are you using a CSR?", certRes.Domain, err.Error())
	}

	jsonBytes, err := json.MarshalIndent(certRes, "", "\t")
	if err != nil {
		log.Fatalf("Unable to marshal CertResource for domain %s\n\t%s", certRes.Domain, err.Error())
	}

	err = ioutil.WriteFile(metaOut, jsonBytes, 0600)
	if err != nil {
		log.Fatalf("Unable to save CertResource for domain %s\n\t%s", certRes.Domain, err.Error())
	}
}

func main() {
	wellKnownDir := filepath.Join(WEBROOT_LOCATION, ".well-known")
	fs := http.StripPrefix("/.well-known/", http.FileServer(http.Dir(wellKnownDir)))
	http.Handle("/.well-known/", fs)
	log.Printf("Serving static files from : %s", wellKnownDir)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/register", registrationHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/", healthHandler)
	httpPort := Getenv("HTTP_PORT", "80")
	log.Printf("HTTP Server listening on port: %s", httpPort)
	http.ListenAndServe(":"+httpPort, nil)
}
