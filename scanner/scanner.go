package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

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

		for _, repo := range result.Items {
			if !repo.Repository.Private {
				repos = append(repos, repo.Repository.FullName)
			}
		}

		if reposOnPage != Config.ReposPerPage {
			complete = true
			break
		}

		page += 1

		if page > 10 {
			fmt.Printf("\t\tFetching over ten pages -> stoping the scanning for %v", name)
			break
		}
	}

	return repos
}

// Fetch all repos that are forks of the template
func fetchByForkOfTemplate() []string {
	var repos []string
	url := fmt.Sprintf("https://api.github.com/repos/MeteorDevelopment/meteor-addon-template/forks?per_page=%v&page=", Config.ReposPerPage)

	var complete bool = false
	var page int = 0
	fmt.Printf("\tFetching fokrs of template\n")
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
			if !repo.Private {
				repos = append(repos, repo.FullName)
			}
		}

		if reposOnPage != Config.ReposPerPage {
			complete = true
			break
		}

		page += 1

		if page > 10 {
			fmt.Println("\t\tFetching over ten pages -> stoping the scanning for forks of template")
			break
		}
	}

	fmt.Printf("\t\tFiltering out repos ending in '-addon-template' -> ")

	var filteredRepos []string

	for _, full_name := range repos {
		cleanedName := strings.TrimSpace(full_name)
		if strings.HasSuffix(strings.ToLower(cleanedName), "-addon-template") {
			continue
		}
		filteredRepos = append(filteredRepos, full_name)
	}
	fmt.Printf("Keeping %v of %v repos\n", len(filteredRepos), len(repos))
	return filteredRepos
}

func Locate() []string {
	url := fmt.Sprintf("https://api.github.com/search/code?q=entrypoints+meteor+extension:json+filename:fabric.mod.json+fork:true+in:file&per_page=%v&page=", Config.ReposPerPage)
	reposByEntryPoint := fetchBySearch("fabric.mod.json", url)
	url = fmt.Sprintf("https://api.github.com/search/code?q=extends+MeteorAddon+language:java+in:file&per_page=%v&page=", Config.ReposPerPage)
	reposByExtendMeteor := fetchBySearch("Extend MeteorAddon", url)
	reposByForkOfTemplate := fetchByForkOfTemplate()

	repos := append(reposByEntryPoint, reposByExtendMeteor...)
	repos = append(repos, reposByForkOfTemplate...)

	return repos
}
