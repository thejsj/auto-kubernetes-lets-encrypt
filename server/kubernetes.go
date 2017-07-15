package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type SecretUpdateTemplate struct {
	Kind       string            `json:"kind"`
	ApiVersion string            `json:"apiVersion"`
	Metadata   map[string]string `json:"metadata"`
	Data       map[string][]byte `json:"data"`
}

func NewSecretUpdate(name string, namespace string, data map[string][]byte) SecretUpdateTemplate {
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
	url := fmt.Sprintf("https://kubernetes/api/v1/namespaces/%s/secrets/%s", namespace, secretName)
	jsonStr, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonStr))
	log.Printf("Request to API: %s", req)
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("Authorization", authorizationHeader)
	client := &http.Client{}
	resp, err := client.Do(req)
	log.Printf("Response from API: %s", resp)
	return nil

}
