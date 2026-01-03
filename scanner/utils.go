package scanner

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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
			fmt.Printf("[%s] Rate limit reached. Waiting %v seconds...",
				apiType, waitTime.Seconds())
			time.Sleep(waitTime + 1*time.Second)
			fmt.Printf(" -> ")
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

	tracker = getRateLimitTracker(resourceType)

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &tracker.Remaining)
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		var timestamp int64
		fmt.Sscanf(reset, "%d", &timestamp)
		tracker.Reset = time.Unix(timestamp, 0)
	}

	rateLimits.mu.Unlock()

	if resp.StatusCode == 403 && tracker.Remaining == 0 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded for %s", resourceType)
	}

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

// CompareMinecraftVersions compares two Minecraft version strings numerically.
//
// Return values:
//
//	-1 if versionA < versionB
//	 0 if versionA == versionB
//	 1 if versionA > versionB
func CompareMinecraftVersions(versionA, versionB string) int {
	partsA := strings.Split(versionA, ".")
	partsB := strings.Split(versionB, ".")

	maxLength := max(len(partsB), len(partsA))

	for index := range maxLength {
		partA := 0
		partB := 0

		if index < len(partsA) {
			partA, _ = strconv.Atoi(partsA[index])
		}

		if index < len(partsB) {
			partB, _ = strconv.Atoi(partsB[index])
		}

		if partA < partB {
			return -1
		}
		if partA > partB {
			return 1
		}
	}

	return 0
}
