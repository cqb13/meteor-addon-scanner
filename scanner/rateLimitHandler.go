package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RateLimit int

const (
	Core   RateLimit = 1
	Search RateLimit = 2
)

func (r RateLimit) String() string {
	switch r {
	case Core:
		return "Core"
	case Search:
		return "Search"
	default:
		return "Unknown"
	}
}

func SleepIfRateLimited(kind RateLimit) error {
	req, err := BuildRequest("https://api.github.com/rate_limit")
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type rateLimitResponse struct {
		Resources struct {
			Core struct {
				Remaining int `json:"remaining"`
				Reset     int `json:"reset"`
			} `json:"core"`
			Search struct {
				Remaining int `json:"remaining"`
				Reset     int `json:"reset"`
			}
		} `json:"resources"`
	}

	var result rateLimitResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	var remaining int
	var reset int

	switch kind {
	case Core:
		remaining = result.Resources.Core.Remaining
		reset = result.Resources.Core.Reset
		break
	case Search:
		remaining = result.Resources.Search.Remaining
		reset = result.Resources.Search.Reset
	}

	if remaining > 0 {
		fmt.Printf("\t[rate limit] %v requests remaining for %v", remaining, kind.String())
		return nil
	}

	var waitTime = max(reset-int(time.Now().Unix()), 25)
	fmt.Printf("\t[rate limit] No requests remaining for %v -> Sleeping until reset (%v Seconds)...", kind.String(), waitTime)
	time.Sleep(time.Duration(waitTime) * time.Second)

	return nil
}
