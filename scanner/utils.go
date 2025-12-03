package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var defaultHeaders http.Header
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

type RateLimitTracker struct {
	Remaining int
	Reset     time.Time
}

var rateLimits struct {
	Search RateLimitTracker
	Core   RateLimitTracker
	mu     sync.Mutex
}

func init() {
	// Initialize with default values
	rateLimits.Search = RateLimitTracker{Remaining: 30, Reset: time.Now()}
	rateLimits.Core = RateLimitTracker{Remaining: 5000, Reset: time.Now()}
}

func detectAPIType(url string) string {
	if strings.Contains(url, "/search/") {
		return "search"
	}
	return "core"
}

func getRateLimitTracker(apiType string) *RateLimitTracker {
	switch apiType {
	case "search":
		return &rateLimits.Search
	case "code_search":
		return &rateLimits.Search
	default:
		return &rateLimits.Core
	}
}

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

func MakeHeadRequest(url string) (int, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func MakeGetRequest(url string) ([]byte, error) {
	// Detect API type from URL (for pre-request check)
	apiType := detectAPIType(url)

	// Check rate limit BEFORE request
	rateLimits.mu.Lock()
	tracker := getRateLimitTracker(apiType)
	if tracker.Remaining <= 1 && time.Now().Before(tracker.Reset) {
		waitTime := time.Until(tracker.Reset)
		if waitTime > 0 {
			fmt.Printf("[%s] Rate limit reached. Waiting %v seconds...\n",
				apiType, waitTime.Seconds())
			time.Sleep(waitTime + 1*time.Second)
		}
	}
	rateLimits.mu.Unlock()

	// Build and execute request
	req, err := BuildRequest(url)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if os.IsTimeout(err) {
			return nil, fmt.Errorf("Request timeout after 30s: %v", err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	// Read and update rate limit AFTER response
	rateLimits.mu.Lock()

	// Read which resource was consumed from response header
	resourceType := resp.Header.Get("X-RateLimit-Resource")
	if resourceType == "" {
		// Fallback to URL detection if header missing
		resourceType = apiType
	}

	// Get the correct tracker for this resource
	tracker = getRateLimitTracker(resourceType)

	// Update from response headers
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &tracker.Remaining)
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		var timestamp int64
		fmt.Sscanf(reset, "%d", &timestamp)
		tracker.Reset = time.Unix(timestamp, 0)
	}

	rateLimits.mu.Unlock()

	// Handle rate limit exceeded - return error to trigger retry
	if resp.StatusCode == 403 && tracker.Remaining == 0 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded for %s", resourceType)
	}

	// Read and return response body
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
