package hello

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testId string
var imageName string
var testNodePort string
var serviceIPaddress string
var serviceName string = "auto-kubernetes-lets-encrypt"
var failed bool = false
var ZONE_ID string = "2fcce5055b9bdafff28874ed2f5a4140"
var DOMAIN string = "jorge.fail"
var CLOUDFLARE_EMAIL string = "jorge.silva@thejsj.com"
var CLOUDFLARE_API_KEY string = os.Getenv("CLOUDFLARE_API_KEY")

type K8sResponse struct {
	Status K8sStatusResponse `json:"status"`
}
type K8sStatusResponse struct {
	LoadBalancer K8sLoadBalancerResponse `json:"loadBalancer"`
}
type K8sLoadBalancerResponse struct {
	Ingress []K8sIngressEntryResponse `json:"ingress"`
}
type K8sIngressEntryResponse struct {
	Ip string `json:"ip"`
}

// #1 It should build an image
func TestBuildingImage(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Start build. Check for commit ENV")
	commit := os.Getenv("BUILD_GIT_COMMIT")
	if len(commit) == 0 {
		t.Fatal("No ENV passed for `BUILD_GIT_COMMIT`. Cannot build image")
	}
	imageName = fmt.Sprintf("quay.io/hiphipjorge/auto-kubernetes-lets-encrypt:%s", commit)
	if os.Getenv("SKIP_BUILD") != "" {
		return
	}
	fullCommand := fmt.Sprintf("docker build -t %s ../server/", imageName)
	t.Logf("Build with command: `%s`", fullCommand)
	err, output := execCommand(fullCommand)
	if err != nil {
		failed = true
		t.Fatalf("Error building image: %s", output)
	}
}

// #2 It should push the image
func TestPushingImage(t *testing.T) {
	if failed || os.Getenv("SKIP_BUILD") != "" {
		t.SkipNow()
	}

	t.Log("Start push. Check for commit ENV")
	fullCommand := fmt.Sprintf("docker push %s", imageName)
	t.Logf("Push with command: `%s`", fullCommand)
	err, output := execCommand(fullCommand)
	if err != nil {
		t.Fatalf("Error push image: %s", output)
	}
}

// #3 It should create a new job
func TestCreatingNamespace(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Create namespace")
	fullCommand := fmt.Sprintf("kubectl create namespace %s", testId)
	t.Logf("Create new kubernetes namespace with command: `%s`", fullCommand)
	err, output := execCommand(fullCommand)
	if err != nil {
		failed = true
		t.Fatalf("Error applying job: %s", output)
	}
}

// #4 It should create a new job
func TestCreatingJob(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Apply kubernetes resources to test namespace")
	err, dstFilaname := copyFileContentsAndReplace("./", "kubernetes-resources.yml", testId, imageName)
	if err != nil {
		failed = true
		t.Fatalf("Failed to execute file replacement", err.Error())
	}
	fullCommand := fmt.Sprintf("kubectl --namespace %s apply -f %s", testId, dstFilaname)
	t.Logf("Update kubernetes with command: `%s`", fullCommand)
	err, output := execCommand(fullCommand)
	if err != nil {
		failed = true
		t.Fatalf("Error applying job: %s", output)
	}
}

// #5 It should wait for a new IP address
func TestCreatingOfIp(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Start to watch for creation of IP address")
	fullCommand := fmt.Sprintf("kubectl --namespace=%s get svc %s -o json", testId, serviceName)
	for {
		time.Sleep(1000 * time.Millisecond)
		t.Log("Fetching IP address...")
		err, output := execCommand(fullCommand)
		if err != nil {
			continue
		}
		res := K8sResponse{}
		err = json.Unmarshal([]byte(output), &res)
		if len(res.Status.LoadBalancer.Ingress) == 0 {
			continue
		}
		if res.Status.LoadBalancer.Ingress[0].Ip == "" {
			continue
		}
		serviceIPaddress = res.Status.LoadBalancer.Ingress[0].Ip
		t.Log("IP address found: %s", serviceIPaddress)
		break
	}
}

// #6 It should create/update DNS entry in cloudflare with IP address
func TestCreatingOfDNSEntry(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Create DNS entry")
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", ZONE_ID)
	jsonStr := fmt.Sprintf("{\"type\":\"A\",\"name\":\"%s.%s\",\"content\": \"%s\"}", testId, DOMAIN, serviceIPaddress)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonStr)))
	req.Header.Set("X-Auth-Email", CLOUDFLARE_EMAIL)
	req.Header.Set("X-Auth-Key", CLOUDFLARE_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		failed = true
		t.Fatalf("Error creating DNS entry: %s, %s", resp.StatusCode, err)
	}
	// TODO: Delete entry
	defer resp.Body.Close()
}

func TestDNSResolution(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Start checking for DNS resolution")
	url := fmt.Sprintf("http://%s.%s", testId, DOMAIN)
	for {
		ips, err := net.LookupIP(url)
		if err != nil {
			t.Log("Error looking up IP for DNS entry: %s", err)
			continue
		}
		if ips[0] != serviceIPaddress {
			t.Log("Error looking up IP for DNS entry: %s", err)
			log.Printf("IP address is different: %s, %s", ips, serviceIPaddress)
			continue
		}
		log.Printf("Ips: %s", ips)
		break
	}
}

// #7 It should wait for the Service to be healthy
func TestHealth(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Start checking for health")
	url := fmt.Sprintf("http://%s.%s", testId, DOMAIN)
	log.Printf("Start! %s", url)
	for {
		time.Sleep(1000 * time.Millisecond)
		log.Printf("request... %s", url)
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != 200 {
			log.Printf("error response... ")
			t.Log("Error making HTTP response: %s", err)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			failed = true
			t.Fatalf("Error Reading body: %s", err)
		}
		log.Println(string(body))
		matched, _ := regexp.MatchString("Healthy.*true", string(body))
		if !matched {
			log.Printf("no match...")
			t.Log("HTTP response did not match")
			continue
		}
		t.Log("HTTP response matched")
		break
	}
}

// #7.2 It should register the user
func TestUserCreation(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Register user")
	url := fmt.Sprintf("http://%s.%s/register", testId, DOMAIN)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		failed = true
		t.Fatalf("Error registering user: %s", err)
	}
	defer resp.Body.Close()
}

// #7.3 It should generate the certs
func TestCertCreation(t *testing.T) {
	if failed {
		t.SkipNow()
	}
	t.Log("Register user")
	url := fmt.Sprintf("http://%s.%s/generate", testId, DOMAIN)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		failed = true
		t.Fatalf("Error registering user: %s", err)
	}
	defer resp.Body.Close()
}

// #8 It should have successfully completed the job
func TestJobCompletion(t *testing.T) {
	// TODO: Check cluster for registration property in secret
	t.SkipNow()
}

// #9 It should have successfully added the registration field
func TestRegistrationCreation(t *testing.T) {
	// TODO: Check cluster for registration property in secret
	t.SkipNow()
}

// #10 It should have successfully added the certs
func TestCertsFound(t *testing.T) {
	// TODO: Check cluster for registration property in secret
	t.SkipNow()
}

func tearDown() {
	// DeleteNamespace()
	DeleteFiles()
}

func DeleteNamespace() error {
	fullCommand := fmt.Sprintf("kubectl delete namespace %s", testId)
	log.Printf("Delete namespace with command: %s", fullCommand)
	err, output := execCommand(fullCommand)
	if err != nil {
		log.Fatal("Error deleting namespace: %s", output)
		return err
	}
	return nil
}

func DeleteFiles() error {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
		return err
	}

	for _, file := range files {
		if strings.Contains(file.Name(), testId) {
			p := path.Join("./", file.Name())
			log.Printf("Delete file: %s", p)
			os.Remove(p)
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testId = strconv.Itoa(int(r.Int31()))
	testNodePort = strconv.Itoa(time.Now().Second()%2767 + 30000)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Print("Tests stopped. Tear down tests.")
			tearDown()
			os.Exit(1)
		}
	}()

	// It should wait for the job to finish
	// It should check for the secret to exist
	// It should check for ingress controller to be updated
	retCode := m.Run()
	tearDown()
	// It should delete DNS entry from cloudflare
	// It should delete service from kubernetes
	// It should delete job from kubernetes
	// It should delete service from kubernetes
	os.Exit(retCode)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func execCommand(cmdString string) (error, string) {
	splitCmd := strings.Split(cmdString, " ")
	cmd := exec.Command(splitCmd[0], splitCmd[1:]...)
	var out, err bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err
	cmdErr := cmd.Run()
	if cmdErr != nil {
		return cmdErr, err.String()
	}
	return nil, out.String()
}

func copyFileContentsAndReplace(dir string, fileName string, testId string, imageName string) (error, string) {
	src := path.Join(dir, fileName)
	newFilename := fmt.Sprintf("%s-%s", testId, fileName)
	dst := path.Join(dir, newFilename)
	read, err := ioutil.ReadFile(src)
	if err != nil {
		return err, dst
	}
	newContents := strings.Replace(string(read), "*IMAGE_NAME*", imageName, -1)
	newContents = strings.Replace(newContents, "*SUBDOMAIN*", testId, -1)
	newContents = strings.Replace(newContents, "*NODE_PORT*", testNodePort, -1)
	err = ioutil.WriteFile(dst, []byte(newContents), 0777)
	if err != nil {
		return err, dst
	}
	return nil, dst
}
