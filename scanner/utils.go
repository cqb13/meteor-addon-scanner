package scanner

import (
	"io"
	"net/http"
)

var defaultHeaders http.Header

const RetryAttempts int = 25

func MakeHeadRequest(url string) (int, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func MakeGetRequest(url string) ([]byte, error) {
	req, err := BuildRequest(url)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func InitDefaultHeaders(token string) {
	defaultHeaders = http.Header{}
	defaultHeaders.Add("Authorization", "token "+token)
	defaultHeaders.Add("Accept", "application/vnd.github.v3+json")
	defaultHeaders.Add("User-Agent", "cqb13/meteor-addon-scanner")
}

func BuildRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, values := range defaultHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	return req, nil
}
