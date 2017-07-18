package main

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log"
	"os"

	"github.com/xenolf/lego/acme"
)

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

func getUser(email string) (LegoUser, error) {
	// Create a user. New accounts need an email and private key to start.
	log.Printf("Get user")
	privateKeyStr := Getenv("LETS_ENCRYPT_USER_CERT", "")
	var user LegoUser
	if privateKeyStr == "" {
		log.Printf("Private key not found for user")
		return user, errors.New("Environment variable `LETS_ENCRYPT_USER_CERT` required")
	}
	log.Printf("Decoding pem key")
	pemKey, _ := pem.Decode([]byte(privateKeyStr))
	key, parseError := x509.ParsePKCS1PrivateKey(pemKey.Bytes)
	if parseError != nil {
		log.Printf("Error parsing key")
		return user, parseError
	}
	log.Printf("Private key found")
	user = LegoUser{
		Email: email,
		key:   key,
	}
	return user, nil
}

func getUserWithRegistration(email string) (LegoUser, error) {
	user, err := getUser(email)
	if err != nil {
		return user, err
	}
	registrationJson := Getenv("LETS_ENCRYPT_USER_REGISTRATION", "")
	log.Printf("Registration JSON: %s", registrationJson)
	if registrationJson == "" {
		log.Printf("Error finding user registration")
		return user, errors.New("Error getting user registration from secret. Register user first.")
	}
	registration := acme.RegistrationResource{}
	err = json.Unmarshal([]byte(registrationJson), &registration)
	if err != nil {
		log.Printf("Error marshaling json for registration: %s", err)
		return user, err
	}
	log.Printf("Populating user with registration: %s", registration)
	user.Registration = &registration
	return user, nil
}

func registerUser(user LegoUser) (LegoUser, error) {
	log.Printf("Register user...")
	caServerHost := Getenv("CA_SERVER", "https://acme-v01.api.letsencrypt.org/directory")
	log.Printf("Creating new user from CA server: %s", caServerHost)
	client, err := acme.NewClient(caServerHost, &user, acme.RSA2048)
	log.Printf("Registering user: %s", user)
	reg, err := client.Register()
	if err != nil {
		log.Printf("Error registering user: %s", err)
		return user, err
	}
	user.Registration = reg
	log.Printf("User registered: %s", reg)
	err = saveRegistration(user)
	if err != nil {
		log.Printf("Error saving user registration", err)
		return user, err
	}
	return user, nil
}

func saveRegistration(user LegoUser) error {
	log.Printf("Save registration...")
	if user.Registration == nil {
		log.Printf("User has no registration", user)
		return errors.New("User has no registration")
	}
	secretName := Getenv("LETS_ENCRYPT_USER_SECRET_NAME", "")
	if secretName == "" {
		return errors.New("Environment variable `SECRET_NAME` required for saving certs")
	}
	updates := make(map[string]string)
	log.Printf("Registratio: %s", *user.Registration)
	registrationUpdatesJson, err := json.Marshal(*user.Registration)
	if err != nil {
		return err
	}
	updates["registration"] = base64.StdEncoding.EncodeToString(registrationUpdatesJson)
	update, err := NewSecretUpdate(secretName, updates)
	if err != nil {
		return err
	}
	err = updateSecret(secretName, update)
	if err != nil {
		return err
	}
	os.Setenv("LETS_ENCRYPT_USER_REGISTRATION", string(registrationUpdatesJson))
	return nil
}
