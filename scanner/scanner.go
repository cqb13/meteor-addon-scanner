package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

var repos = make(map[string]bool)

const reposPerPage int = 100

func fetchBySearch(name string, url string) {
	var attempts int = RetryAttempts

	var complete bool = false
	var page int = 1
	fmt.Printf("\tFetching based on %v\n", name)
	for {
		if complete == true {
			break
		}

		if attempts == 0 {
			fmt.Printf("Failed to make search request\n")
			os.Exit(1)
		}

		fmt.Printf("\t")
		SleepIfRateLimited(Search, false)
		fmt.Printf("\t\tFetching Page %v -> ", page)
		bytes, err := MakeGetRequest(fmt.Sprintf("%s%v", url, page))
		if err != nil {
			attempts -= 1
			continue
		}

		if strings.HasSuffix(string(bytes), "\"status\":\"403\"}") {
			fmt.Printf("Rate Limited -> Sleeping for 60 seconds...\n")
			time.Sleep(60 * time.Second)
			continue
		}

		type githubPages struct {
			Items []struct {
				Repository struct {
					FullName string `json:"full_name"`
					Private  bool   `json:"private"`
				}
			} `json:"items"`
		}

		var result githubPages

		err = json.Unmarshal(bytes, &result)
		if err != nil {
			fmt.Printf("Failed to parse JSON\n")
			os.Exit(1)
		}

		var reposOnPage int = len(result.Items)

		fmt.Printf("Found %v Repositories\n", reposOnPage)

		for _, repo := range result.Items {
			_, ok := repos[strings.ToLower(repo.Repository.FullName)]

			if !repo.Repository.Private && !ok && !strings.HasSuffix(strings.ToLower(repo.Repository.FullName), "-addon-template") {
				repos[repo.Repository.FullName] = false
			}
		}

		if reposOnPage == 0 {
			complete = true
			break
		}

		page += 1

		if page > 10 {
			fmt.Printf("\t\tFetching over ten pages -> stoping the scanning for %v", name)
			break
		}
	}
}

// Fetch all repos that are forks of the template
func fetchByForkOfTemplate() {
	var attempts int = RetryAttempts
	url := fmt.Sprintf("https://api.github.com/repos/MeteorDevelopment/meteor-addon-template/forks?per_page=%v&page=", reposPerPage)

	var complete bool = false
	var page int = 1
	fmt.Printf("\tFetching fokrs of template\n")
	for {
		if complete == true {
			break
		}

		if attempts == 0 {
			fmt.Printf("Failed to make search request\n")
			os.Exit(1)
		}
		fmt.Printf("\t")
		SleepIfRateLimited(Search, false)
		fmt.Printf("\t\tFetching Page %v -> ", page)
		bytes, err := MakeGetRequest(fmt.Sprintf("%s%v", url, page))
		if err != nil {
			attempts -= 1
			continue
		}

		if strings.HasSuffix(string(bytes), "\"status\":\"403\"}") {
			fmt.Printf("Rate Limited -> Sleeping for 60 seconds...\n")
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

		var reposOnPage int = len(result)

		fmt.Printf("Found %v Repositories\n", reposOnPage)

		for _, repo := range result {
			_, ok := repos[strings.ToLower(repo.FullName)]

			if !repo.Private && !ok && !strings.HasSuffix(strings.ToLower(repo.FullName), "-addon-template") {
				repos[repo.FullName] = false
			}
		}

		if reposOnPage == 0 {
			complete = true
			break
		}

		page += 1

		if page > 10 {
			fmt.Println("\t\tFetching over ten pages -> stoping the scanning for forks of template")
			break
		}
	}
}

func Locate(verifiedAddons []string) map[string]bool {
	for _, addon := range verifiedAddons {
		repos[addon] = true
	}

	url := fmt.Sprintf("https://api.github.com/search/code?q=entrypoints+meteor+extension:json+filename:fabric.mod.json+fork:true+in:file&per_page=%v&page=", reposPerPage)
	fetchBySearch("fabric.mod.json", url)
	url = fmt.Sprintf("https://api.github.com/search/code?q=extends+MeteorAddon+language:java+in:file&per_page=%v&page=", reposPerPage)
	fetchBySearch("Extend MeteorAddon", url)
	fetchByForkOfTemplate()

	return repos
}
