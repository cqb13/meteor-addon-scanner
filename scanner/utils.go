package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var defaultHeaders http.Header

const RetryAttempts int = 25

type forkedRepository struct {
	Parent struct {
		PushedAt string `json:"pushed_at"`
	} `json:"parent"`
}

type ForkValidationResult int

const (
	Valid ForkValidationResult = iota
	InvalidChildTooOld
	InvalidParentTooRecent
)

func ValidateForkedVerifiedAddons(addon Addon) (ForkValidationResult, error) {
	currentTime := time.Now()

	childUpdateTime, err := time.Parse(time.RFC3339, addon.Repo.LastUpdate)
	if err != nil {
		return 0, err
	}

	// Reject if the child hasn't been updated in 6 months
	if childUpdateTime.AddDate(0, 6, 0).Before(currentTime) {
		return InvalidChildTooOld, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s", addon.Repo.Id)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return 0, err
	}

	var repo forkedRepository
	err = json.Unmarshal(bytes, &repo)
	if err != nil {
		return 0, err
	}

	parentUpdateTime, err := time.Parse(time.RFC3339, repo.Parent.PushedAt)
	if err != nil {
		return 0, err
	}

	// Reject if the parent has been updated within 6 months of now
	if parentUpdateTime.AddDate(0, 6, 0).After(currentTime) {
		return InvalidParentTooRecent, nil
	}

	return Valid, nil
}

func Dedupe(input []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, str := range input {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}

	return result
}

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
