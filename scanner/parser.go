package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
)

type Addon struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	McVersion    string   `json:"mc_version"`
	Authors      []string `json:"authors"`
	Features     []string `json:"features"`
	FeatureCount int      `json:"feature_count"`
	Verified     bool     `json:"verified"`
	Repo         Repo     `json:"repo"`
	Links        Links    `json:"links"`
}

type Repo struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	Fork         bool   `json:"fork"`
	Stars        int    `json:"stars"`
	Downloads    int    `json:"downloads"`
	LastUpdate   string `json:"last_update"`
	CreationDate string `json:"creation_date"`
}

type Links struct {
	Github   string `json:"github"`
	Download string `json:"download"`
	Discord  string `json:"discord"`
	Homepage string `json:"homepage"`
	Icon     string `json:"icon"`
}

// matches patterns like `add(new SomeFeatureName(...))`
// and captures the feature name (e.g., "SomeFeatureName")
var featureRegex = regexp.MustCompile(`(?:add\(new )([^(]+)(?:\([^)]*)\)\)`)

// matches Discord invite links, supporting various domains
// and formats (e.g., "https://discord.gg/abc123", "discord.com/invite/abc")
var inviteRegex = regexp.MustCompile(`((?:https?:\/\/)?(?:www\.)?(?:discord\.(?:gg|io|me|li|com)|discordapp\.com/invite|dsc\.gg)/[a-zA-Z0-9\-\/]+)`)

type repository struct {
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Stars         int    `json:"stargazers_count"`
	DefaultBranch string `json:"default_branch"`
	HtmlUrl       string `json:"html_url"`
	PushedAt      string `json:"pushed_at"`
	CreatedAt     string `json:"created_at"`
	Fork          bool   `json:"fork"`
	Archived      bool   `json:"archived"`
	Homepage      string `json:"homepage"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type fabric struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Authors     []string `json:"authors"`
	Icon        string   `json:"icon"`
	Entrypoints struct {
		Meteor []string `json:"meteor"`
	} `json:"entrypoints"`
}

type release struct {
	Assets []struct {
		Name      string `json:"name"`
		Url       string `json:"browser_download_url"`
		Downloads int    `json:"download_count"`
	} `json:"assets"`
}

func getRepo(fullName string) (*repository, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v", fullName)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", err
	}

	var repo repository

	err = json.Unmarshal(bytes, &repo)
	if err != nil {
		return nil, "", err
	}

	return &repo, string(bytes), nil
}

func getFabricModJson(fullName string, defaultBranch string) (*fabric, string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/resources/fabric.mod.json", fullName, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", err
	}

	var fabricModJson fabric

	err = json.Unmarshal(bytes, &fabricModJson)
	if err != nil {
		return nil, "", err
	}

	return &fabricModJson, string(bytes), nil
}

// https://api.github.com/repos/{name}/releases
func getReleaseDetails(fullName string) (string, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/releases", fullName)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", 0, err
	}

	var releases []release

	err = json.Unmarshal(bytes, &releases)
	if err != nil {
		return "", 0, err
	}

	var downloadCount int = 0
	var downloadUrl string = ""

	for _, release := range releases {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)

			if strings.HasSuffix(name, "-dev.jar") || strings.HasSuffix(name, "-sources.jar") {
				continue
			}

			if strings.HasSuffix(name, ".jar") {
				downloadCount += asset.Downloads
				if downloadUrl == "" {
					downloadUrl = asset.Url
				}
				break
			}
		}
	}

	if downloadUrl == "" {
		fmt.Println("\t\tMissing release")
	}

	return downloadUrl, downloadCount, nil
}

func getIcon(fullName string, defaultBranch string, icon string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/resources/%v", fullName, defaultBranch, icon)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	if string(bytes) == "404: Not Found" {
		fmt.Println("\t\tMissing icon")
		return "", nil
	}

	return url, nil
}

func findDiscordServer(fullName string, defaultBranch string, repoStr string, fabricStr string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/README.md", fullName, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	readme := string(bytes)

	matches := inviteRegex.FindAllString(readme, -1)
	matches = append(matches, inviteRegex.FindAllString(fabricStr, -1)...)
	matches = append(matches, inviteRegex.FindAllString(repoStr, -1)...)

	for _, invite := range matches {
		if !regexp.MustCompile(`^https?://`).MatchString(invite) {
			invite = "https://" + invite
		}
		status, err := MakeHeadRequest(invite)
		if err == nil && status != 404 {
			return invite, nil
		}
	}

	return "", nil
}

func findFeatures(fullName string, defaultBranch string, entrypoint string) ([]string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v.java", fullName, defaultBranch, strings.ReplaceAll(entrypoint, ".", "/"))
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, err
	}

	var features []string

	features = append(features, featureRegex.FindAllString(string(bytes), -1)...)

	for i := range features {
		features[i] = strings.Replace(features[i], "add(new ", "", -1)
		features[i] = strings.Replace(features[i], "())", "", -1)
	}

	return features, nil
}

func findVersion(fullName string, defaultBranch string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/gradle.properties", fullName, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	var version string = ""
	for line := range strings.SplitSeq(string(bytes), "\n") {
		if strings.HasPrefix(line, "minecraft_version=") {
			version = strings.TrimSpace(strings.Replace(line, "minecraft_version=", "", 1))
		}
	}

	return version, nil
}

func parseRepo(fullName string, number int, total int) (*Addon, error) {
	fmt.Printf("\tParsing %v, %v/%v\n", fullName, number, total)
	fmt.Printf("\t")
	SleepIfRateLimited(Core)
	repo, repoStr, err := getRepo(fullName)
	if err != nil {
		return nil, err
	}

	fabricModJson, fabricStr, err := getFabricModJson(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	// ensure meteor entrypoint is present
	if len(fabricModJson.Entrypoints.Meteor) == 0 {
		return nil, fmt.Errorf("Missing meteor entrypoint")
	}

	// ensure a description is present
	description := repo.Description
	if description == "" {
		description = fabricModJson.Description
	}
	if description == "" {
		fmt.Println("\t\tMissing description")
	}

	// find authors from fabric.mod.json or from github username
	var authors []string
	if len(fabricModJson.Authors) == 0 {
		authors = append(authors, repo.Owner.Login)
	} else {
		authors = fabricModJson.Authors
	}

	downloadUrl, downloadCount, err := getReleaseDetails(fullName)
	if err != nil {
		return nil, err
	}

	icon, err := getIcon(fullName, repo.DefaultBranch, fabricModJson.Icon)
	if err != nil {
		return nil, err
	}

	invite, err := findDiscordServer(fullName, repo.DefaultBranch, repoStr, fabricStr)
	if err != nil {
		return nil, err
	}

	features, err := findFeatures(fullName, repo.DefaultBranch, fabricModJson.Entrypoints.Meteor[0])
	if err != nil {
		return nil, err
	}

	version, err := findVersion(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	site := repo.Homepage

	// prevent the homepage being a discord invite
	if inviteRegex.MatchString(site) {
		site = ""
	}

	addon := Addon{
		Name:         fabricModJson.Name,
		Description:  fabricModJson.Description,
		McVersion:    version,
		Authors:      authors,
		Features:     features,
		FeatureCount: len(features),
		Verified:     false,
		Repo: Repo{
			Id:           fullName,
			Owner:        repo.Owner.Login,
			Name:         repo.Name,
			Archived:     repo.Archived,
			Fork:         repo.Fork,
			Stars:        repo.Stars,
			Downloads:    downloadCount,
			LastUpdate:   repo.PushedAt,
			CreationDate: repo.CreatedAt,
		},
		Links: Links{
			Github:   repo.HtmlUrl,
			Download: downloadUrl,
			Discord:  invite,
			Icon:     icon,
			Homepage: site,
		},
	}

	fmt.Printf("\tFinished Parsing %v, %v/%v\n", fullName, number, total)
	return &addon, nil
}

func ParseRepos(verifiedAddonsPath string, repos []string) []*Addon {
	var total int = len(repos)
	var addons []*Addon

	file, err := os.Open(verifiedAddonsPath)
	if err != nil {
		fmt.Printf("Failed to load verified addons: %v\n", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Failed to read verified addons: %v\n", err)
	}

	var verifiedAddons []string

	for line := range strings.SplitSeq(string(bytes), "\n") {
		verifiedAddons = append(verifiedAddons, line)
	}

	for i, fullName := range repos {
		addon, err := parseRepo(fullName, i+1, total)

		if err != nil {
			fmt.Printf("\tFailed to parse %v: %v\n", fullName, err)
			continue
		}

		if slices.Contains(verifiedAddons, addon.Repo.Id) {
			addon.Verified = true
		}

		addons = append(addons, addon)
	}

	fmt.Printf("Found %v valid addons out of %v repositories\n", len(addons), len(repos))

	return addons
}
