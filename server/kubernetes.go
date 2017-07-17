package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type SecretUpdateTemplate struct {
	Kind       string            `json:"kind"`
	ApiVersion string            `json:"apiVersion"`
	Metadata   map[string]string `json:"metadata"`
	Data       map[string]string `json:"data"`
}

func NewSecretUpdate(name string, namespace string, data map[string]string) SecretUpdateTemplate {
	metadata := make(map[string]string)
	metadata["name"] = name
	metadata["namespace"] = namespace
	return SecretUpdateTemplate{
		Kind:       "Secret",
		ApiVersion: "v1",
		Metadata:   metadata,
		Data:       data,
	}
}

var NAMESPACE_LOCATION = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
var TOKEN_LOCATION = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func getNamespace() (string, error) {
	log.Printf("Looking for kuberentes namespace in: %s", NAMESPACE_LOCATION)
	fileData, err := ioutil.ReadFile(NAMESPACE_LOCATION)
	var namespace string
	if err != nil {
		return namespace, err
	}
	namespace = string(fileData)
	return namespace, nil
}

func getToken() (string, error) {
	log.Printf("Looking for kuberentes token in: %s", TOKEN_LOCATION)
	fileData, err := ioutil.ReadFile(TOKEN_LOCATION)
	var namespace string
	if err != nil {
		return namespace, err
	}
	namespace = string(fileData)
	return namespace, nil
}

func updateSecret(secretName string, update SecretUpdateTemplate) error {
	namespace, err := getNamespace()
	if err != nil {
		return err
	}
	token, err := getToken()
	if err != nil {
		return err
	}
	kubernestsHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	if kubernestsHost == "" {
		return errors.New("No `KUBERNETES_SERVICE_HOST` defined")
	}
	url := fmt.Sprintf("https://%s/api/v1/namespaces/%s/secrets/%s", kubernestsHost, namespace, secretName)
	jsonStr, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonStr))
	log.Printf("Request to API: %s", req)
	log.Printf("Test 0.0")
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	log.Printf("Test 0")
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")
	log.Printf("Test 0.1")
	req.Header.Set("Authorization", authorizationHeader)
	log.Printf("Test 1")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	log.Printf("Test 2.0")
	client := &http.Client{Transport: tr}
	log.Printf("Test 2.1")
	resp, err := client.Do(req)
	log.Printf("Test 2.2")
	if err != nil {
		return err
	}
	log.Printf("Test 3")
	defer resp.Body.Close()
	log.Printf("Test 3.1")
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Test 4")
	log.Printf("Response from API: %s, %s", resp.StatusCode, string(body))
	if resp.StatusCode != 200 {
		return fmt.Errorf("User registration did not return 200 (Status Code: %s): %s", resp.StatusCode, string(body))
	}
	return nil

}
