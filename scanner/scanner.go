package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

var reposPerPage = 0

func SetReposPerPage(repos int) {
	reposPerPage = repos
}

// Fetch all repos based on a search
func fetchBySearch(name string, url string) []string {
	var repos []string

	var complete bool = false
	var page int = 0
	fmt.Printf("\tFetching based on %v\n", name)
	for {
		if complete == true {
			break
		}
		fmt.Printf("\t")
		SleepIfRateLimited(Search)
		fmt.Printf("\t\tFetching Page %v -> ", page)
		bytes, err := MakeGetRequest(fmt.Sprintf("%s%v", url, page))
		if err != nil {
			fmt.Printf("Failed to make search request\n")
			os.Exit(1)
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

		if reposOnPage != reposPerPage {
			complete = true
			break
		}

		for _, repo := range result.Items {
			if !repo.Repository.Private {
				repos = append(repos, repo.Repository.FullName)
			}
		}

		page += 1

		if page > 10 {
			fmt.Println("\t\tFetching over ten pages -> stoping the scanning for fabric.mod.json")
			break
		}
	}

	return repos

}

// Fetch all repos that are forks of the template
func fetchByForkOfTemplate() []string {
	var repos []string

	return repos

}

func Locate() []string {
	fmt.Println(reposPerPage)
	url := fmt.Sprintf("https://api.github.com/search/code?q=entrypoints+meteor+extension:json+filename:fabric.mod.json+fork:true+in:file&per_page=%v&page=", reposPerPage)
	reposByEntryPoint := fetchBySearch("fabric.mod.json", url)
	url = fmt.Sprintf("https://api.github.com/search/code?q=extends+MeteorAddon+language:java+in:file&per_page=%v&page=", reposPerPage)
	reposByExtendMeteor := fetchBySearch("Extend MeteorAddon", url)
	reposByForkOfTemplate := fetchByForkOfTemplate()

	repos := append(reposByEntryPoint, reposByExtendMeteor...)
	repos = append(repos, reposByForkOfTemplate...)

	return repos
}
