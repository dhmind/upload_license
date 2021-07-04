package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	login := flag.String("login", "", "User for login from UI.")
	password := flag.String("password", "", "Password to user.")
	ip := flag.String("ip", "", "The host where you want to upload license.")
	path := flag.String("path", "", "Path to license.")

	flag.Parse()

	if ip == nil || *ip == "" {
		log.Fatal("ip is empty")
	}
	if login == nil || *login == "" {
		log.Fatal("login is empty")
	}
	if password == nil || *password == "" {
		log.Fatal("password is empty")
	}
	if path == nil || *path == "" {
		log.Fatal("path is empty")
	}

	err := uploadLicenseToAllHosts(*ip, *login, *password, *path)
	if err != nil {
		log.Fatal(err)
	}
}

func uploadLicenseToAllHosts(ip, user, password, licenseFilepath string) (err error) {
	client := NewApiClient(ip)

	err = client.Authorize(user, password)
	if err != nil {
		return fmt.Errorf("can't authorize: %w", err)
	}

	var hostIDs []string
	hostIDs, err = client.ListHosts()
	if err != nil {
		return fmt.Errorf("host id is empty: %w", err)
	}
	log.Println("Uploading to hosts...")
	for _, hostID := range hostIDs {
		log.Println(hostID)
		err = client.UploadLicenseToHost(hostID, licenseFilepath)
		if err != nil {
			return fmt.Errorf("can't upload license: %w", err)
		}
	}

	return nil
}

type ApiClient struct {
	httpClient  *http.Client
	accessToken string
	ip          string
}

func NewApiClient(ip string) *ApiClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec we dont knoiw how to manage certificates in bi.zone
		},
	}
	return &ApiClient{
		httpClient: &http.Client{Transport: tr,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("stopped after 10 redirects")
				}
				return nil
			},
			Timeout: time.Second * 10},
		ip: "https://" + ip + ":9993/api/v1",
	}
}

func (apiClient *ApiClient) Authorize(username, password string) error {
	formData := url.Values{
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
	}
	resp, err := apiClient.httpClient.PostForm(apiClient.ip+"/oauth2/token", formData)
	if err != nil {
		return fmt.Errorf("can't send request with PostForm: %w", err)
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:expires_in`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type`
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected statuscode: %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("can't decode response: %w", err)
	}
	if result.AccessToken == "" {
		return fmt.Errorf("access token is empty")
	}
	apiClient.accessToken = result.AccessToken
	log.Println("authorize success")
	return nil
}

func (apiClient *ApiClient) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, apiClient.ip+url, body)
	if err != nil {
		return nil, fmt.Errorf("can't create new request:%w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiClient.accessToken)
	return req, nil
}

func (apiClient *ApiClient) ListHosts() ([]string, error) {
	var hostIDs []string

	req, err := apiClient.NewRequest("GET", "/hosts", nil)
	if err != nil {
		return nil, fmt.Errorf("can't GET request %w", err)
	}
	res, err := apiClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't authorize %w", err)
	}
	var result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("can't decode hosts json %w", err)
	}
	for i := range result {
		hostIDs = append(hostIDs, result[i].ID)

	}
	log.Println("slice of hostsID", hostIDs)

	defer res.Body.Close()

	return hostIDs, nil
}

func (apiClient *ApiClient) UploadLicenseToHost(hostID string, licenseFilename string) error {
	file, err := os.Open(licenseFilename)
	if err != nil {
		return fmt.Errorf("can't open file %s %w", licenseFilename, err)
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("can't read file %s %w", licenseFilename, err)
	}
	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("can't read file stats %s %w", licenseFilename, err)
	}
	defer file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("license", fi.Name())
	if err != nil {
		return fmt.Errorf("can't write license %w", err)
	}

	part.Write(fileContents)
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("can't close file %w", err)
	}

	request, err := apiClient.NewRequest("POST", "/hosts/"+hostID+"/licenses", body)
	if err != nil {
		return fmt.Errorf("can't send POST request %w", err)
	}
	request.Close = true
	request.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := apiClient.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("can't run request %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Println("Error uploading license")
		log.Println("StatusCode")
		log.Println(res.StatusCode)
		return nil
	}

	log.Println("License successfully uploaded")
	return nil
}
