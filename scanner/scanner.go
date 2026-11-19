package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type InvalidAddon struct {
	Name    string         `json:"name"`
	URL     string         `json:"url"`
	Reason  string         `json:"reason"`
	Details map[string]any `json:"details,omitempty"`
}

var mu sync.RWMutex
var repos = make(map[string]struct{})

const reposPerPage int = 100

func fetchBySearch(name string, url string) {
	attempts := RetryAttempts

	page := 1
	for {
		if attempts == 0 {
			fmt.Printf("Failed to make search request\n")
			os.Exit(1)
		}

		bytes, err := MakeGetRequest(fmt.Sprintf("%s%v", url, page))
		if err != nil {
			fmt.Printf("Error: %v (attempt %d/%d)\n", err, RetryAttempts-attempts+1, RetryAttempts)
			attempts -= 1
			continue
		}

		if strings.HasSuffix(string(bytes), "\"status\":\"403\"}") {
			fmt.Printf("\tRate Limited -> Sleeping for 60 seconds...\n")
			time.Sleep(60 * time.Second)
			continue
		}

		type githubPages struct {
			Items []struct {
				// search/repositories
				FullName string `json:"full_name"`
				Private  bool   `json:"private"`

				// search/code
				Repository *struct {
					FullName string `json:"full_name"`
					Private  bool   `json:"private"`
				} `json:"repository"`
			} `json:"items"`
		}

		var result githubPages

		err = json.Unmarshal(bytes, &result)
		if err != nil {
			fmt.Printf("Failed to parse JSON: %s\n")
			os.Exit(1)
		}

		var reposOnPage int = len(result.Items)

		fmt.Printf("\tFound %v Repositories on Page %d of '%s'\n", reposOnPage, page, name)

		for _, item := range result.Items {
			var full string
			var priv bool

			if item.Repository != nil {
				// code search
				full = item.Repository.FullName
				priv = item.Repository.Private
			} else {
				// repo search
				full = item.FullName
				priv = item.Private
			}

			if full == "" {
				// should never happen
				continue
			}

			lower := strings.ToLower(full)

			mu.Lock()
			if !priv && !strings.HasSuffix(lower, "-addon-template") {
				if _, ok := repos[lower]; !ok {
					repos[lower] = struct{}{}
				}
			}
			mu.Unlock()
		}

		if reposOnPage == 0 {
			break
		}

		page += 1

		if page > 10 {
			fmt.Printf("\tFetching over ten pages -> stoping the scanning for %v\n", name)
			break
		}
	}
}

// Fetch all repos that are forks of the template
func fetchByForkOfTemplate() {
	attempts := RetryAttempts
	url := fmt.Sprintf("https://api.github.com/repos/MeteorDevelopment/meteor-addon-template/forks?per_page=%v&page=", reposPerPage)

	page := 1
	for {
		if attempts == 0 {
			fmt.Printf("Failed to make search request\n")
			os.Exit(1)
		}
		bytes, err := MakeGetRequest(fmt.Sprintf("%s%v", url, page))
		if err != nil {
			fmt.Printf("\tError: %v (attempt %d/%d)\n", err, RetryAttempts-attempts+1, RetryAttempts)
			attempts -= 1
			continue
		}

		if strings.HasSuffix(string(bytes), "\"status\":\"403\"}") {
			fmt.Printf("\tRate Limited -> Sleeping for 60 seconds...\n")
			time.Sleep(60 * time.Second)
			continue
		}

		type Repository struct {
			FullName string `json:"full_name"`
			Private  bool   `json:"private"`
		}

		var result []Repository

		err = json.Unmarshal(bytes, &result)
		if err != nil {
			fmt.Printf("Failed to parse JSON\n")
			os.Exit(1)
		}

		reposOnPage := len(result)

		fmt.Printf("\tFound %v Repositories on Page %d of 'forks of template'\n", reposOnPage, page)

		for _, repo := range result {
			_, ok := repos[strings.ToLower(repo.FullName)]

			if !repo.Private && !ok && !strings.HasSuffix(strings.ToLower(repo.FullName), "-addon-template") {
				mu.Lock()
				repos[repo.FullName] = struct{}{}
				mu.Unlock()
			}
		}

		if reposOnPage == 0 {
			break
		}

		page += 1

		if page > 10 {
			fmt.Println("\tFetching over ten pages -> stoping the scanning for forks of template")
			break
		}
	}
}

func Locate(verifiedAddons []string) map[string]struct{} {
	for _, addon := range verifiedAddons {
		repos[strings.ToLower(addon)] = struct{}{}
	}

	searchQueries := map[string]string{
		"meteor-entrypoint":                   fmt.Sprintf("https://api.github.com/search/code?q=entrypoints+meteor+extension:json+filename:fabric.mod.json+fork:true+in:file&per_page=%v&page=", reposPerPage),
		"Extend MeteorAddon":                  fmt.Sprintf("https://api.github.com/search/code?q=extends+MeteorAddon+language:java+in:file&per_page=%v&page=", reposPerPage),
		"meteor-addon topic":                  fmt.Sprintf("https://api.github.com/search/repositories?q=topic:meteor-addon&per_page=%d&page=", reposPerPage),
		"meteor-client-addon topic":           fmt.Sprintf("https://api.github.com/search/repositories?q=topic:meteor-client-addon&per_page=%d&page=", reposPerPage),
		"meteor-addon in name or description": fmt.Sprintf("https://api.github.com/search/repositories?q=meteor-addon+in:name,description&per_page=%d&page=", reposPerPage),
	}

	var wg sync.WaitGroup

	for name, query := range searchQueries {
		wg.Go(func() {
			fetchBySearch(name, query)
		})
	}

	wg.Go(func() {
		fetchByForkOfTemplate()
	})

	wg.Wait()

	return repos
}
