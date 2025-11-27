package scanner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Tag int

const (
	PvP Tag = iota
	Utility
	Theme
	Render
	Movement
	Building
	World
	Misc
	QoL
	Exploit
	Fun
	Automation
)

func (t Tag) String() string {
	switch t {
	case PvP:
		return "PvP"
	case Utility:
		return "Utility"
	case Theme:
		return "Theme"
	case Render:
		return "Render"
	case Movement:
		return "Movement"
	case Building:
		return "Building"
	case World:
		return "World"
	case Misc:
		return "Misc"
	case QoL:
		return "QoL"
	case Exploit:
		return "Exploit"
	case Fun:
		return "Fun"
	case Automation:
		return "Automation"
	default:
		return "Unknown"
	}
}

var validTags = map[string]Tag{
	"pvp":        PvP,
	"utility":    Utility,
	"theme":      Theme,
	"render":     Render,
	"movement":   Movement,
	"building":   Building,
	"world":      World,
	"misc":       Misc,
	"qol":        QoL,
	"exploit":    Exploit,
	"fun":        Fun,
	"automation": Automation,
}

type Addon struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	McVersion   string   `json:"mc_version"`
	Authors     []string `json:"authors"`
	Features    Features `json:"features"`
	Verified    bool     `json:"verified"`
	Repo        Repo     `json:"repo"`
	Links       Links    `json:"links"`
	Custom      Custom   `json:"custom"`
}

type Custom struct {
	Description       string   `json:"description"`
	Tags              []string `json:"tags"`
	SupportedVersions []string `json:"supported_versions"`
	Icon              string   `json:"icon"`
	Discord           string   `json:"discord"`
	Homepage          string   `json:"homepage"`
}

type Features struct {
	Modules      []string `json:"modules"`
	Commands     []string `json:"commands"`
	HudElements  []string `json:"hud_elements"`
	FeatureCount int      `json:"feature_count"`
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

// matches Discord invite links, supporting various domains
// and formats (e.g., "https://discord.gg/abc123", "discord.com/invite/abc")
var inviteRegex = regexp.MustCompile(`((?:https?:\/\/)?(?:www\.)?(?:discord\.(?:gg|io|me|li|com)|discordapp\.com/invite|dsc\.gg)/[a-zA-Z0-9\-\/]+)`)

var mcVersionRegex = regexp.MustCompile(`^1\.\d+(\.\d+)?$`)

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

func validTag(tag string) (bool, string) {
	realTag, exists := validTags[strings.ToLower(tag)]
	return exists, realTag.String()
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

	if string(bytes) == "404: Not Found" {
		return nil, "", fmt.Errorf("fabric.mod.json not found in expected location")
	}

	var fabricModJson fabric

	err = json.Unmarshal(bytes, &fabricModJson)
	if err != nil {
		return nil, "", err
	}

	return &fabricModJson, string(bytes), nil
}

func getCustomProperties(fullName string, defaultBranch string) (*Custom, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/meteor-addon-list.json", fullName, defaultBranch)

	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, err
	}

	var customData Custom

	if string(bytes) == "404: Not Found" {
		return &customData, nil
	}

	err = json.Unmarshal(bytes, &customData)
	if err != nil {
		return nil, err
	}

	// Validate all supported versions
	validVersions := make([]string, 0, len(customData.SupportedVersions))
	for _, v := range customData.SupportedVersions {
		v = strings.TrimSpace(v)
		if mcVersionRegex.MatchString(v) {
			validVersions = append(validVersions, v)
		}
	}
	customData.SupportedVersions = validVersions

	var validTags []string

	for _, tag := range customData.Tags {
		exists, realTag := validTag(tag)

		if exists {
			validTags = append(validTags, realTag)
		}
	}

	customData.Tags = validTags

	return &customData, nil
}

// https://api.github.com/repos/{name}/releases
func getReleaseDetails(fullName string) (string, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/releases?per_page=100&page=", fullName)

	var downloadCount int = 0
	var downloadUrl string = ""

	var complete bool = false
	var page int = 1

	for {
		if complete {
			break
		}

		SleepIfRateLimited(Core, true)

		bytes, err := MakeGetRequest(fmt.Sprintf("%v%v", url, page))
		if err != nil {
			return "", 0, err
		}

		var releases []release

		err = json.Unmarshal(bytes, &releases)
		if err != nil {
			return "", 0, err
		}

		if len(releases) == 0 {
			complete = true
			break
		}

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

		page += 1
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

func splitCamelCase(input string) string {
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	return re.ReplaceAllString(input, "$1 $2")
}

func detectVariable(source string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(source)
	if len(match) >= 2 {
		return match[1]
	}
	return "" // fallback to default pattern
}

func findFeatures(fullName string, defaultBranch string, entrypoint string) (Features, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v.java",
		fullName, defaultBranch, strings.ReplaceAll(entrypoint, ".", "/"))
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return Features{}, err
	}

	source := string(bytes)

	// Detect Module and HUD variable names. Why are people so extra and do this? I'm looking at you, rejects.
	moduleVar := detectVariable(source, `(?m)\bModules\s+(\w+)\s*=\s*(Modules\.get\(\)|Systems\.get\(Modules\.class\));`)
	hudVar := detectVariable(source, `(?m)\bHud\s+(\w+)\s*=\s*(Hud\.get\(\)|Systems\.get\(Hud\.class\));`)

	modulePattern := `(?m)(Modules\.get\(\)|Systems\.get\(Modules\.class\)`
	hudPattern := `(?m)(Hud\.get\(\)|Systems\.get\(Hud\.class\)`
	if moduleVar != "" {
		modulePattern += `|` + regexp.QuoteMeta(moduleVar)
	}
	if hudVar != "" {
		hudPattern += `|` + regexp.QuoteMeta(hudVar)
	}
	modulePattern += `)\.add\(new (\w+)\(\)\);`
	hudPattern += `)\.register\((\w+)\.INFO\);`

	moduleRegex := regexp.MustCompile(modulePattern)
	hudRegex := regexp.MustCompile(hudPattern)
	commandRegex := regexp.MustCompile(`(?m)Commands\.add\(new (\w+)\(\)\);`)

	var modules, hudElements, commands []string

	for _, match := range moduleRegex.FindAllStringSubmatch(source, -1) {
		modules = append(modules, splitCamelCase(match[2]))
	}
	for _, match := range hudRegex.FindAllStringSubmatch(source, -1) {
		hudElements = append(hudElements, splitCamelCase(match[2]))
	}
	for _, match := range commandRegex.FindAllStringSubmatch(source, -1) {
		commands = append(commands, splitCamelCase(match[1]))
	}

	return Features{
		Modules:      modules,
		Commands:     commands,
		HudElements:  hudElements,
		FeatureCount: len(modules) + len(hudElements) + len(commands),
	}, nil
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
			break
		}
	}

	if version == "" {
		version, err = findVersionInGradleCatalog(fullName, defaultBranch)
		if err != nil {
			return "", err
		}
	}

	return version, nil
}

func findVersionInGradleCatalog(fullName string, defaultBranch string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/gradle/libs.versions.toml", fullName, defaultBranch)

	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	var version string = ""
	for line := range strings.SplitSeq(string(bytes), "\n") {
		if strings.HasPrefix(line, "minecraft = ") {
			version = strings.TrimSpace(strings.Replace(line, "minecraft = ", "", 1))
			version = strings.ReplaceAll(version, "\"", "")
			break
		}
	}

	return version, nil
}

func parseRepo(fullName string) (*Addon, error) {
	SleepIfRateLimited(Core, true)
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

	customProperties, err := getCustomProperties(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	addon := Addon{
		Name:        fabricModJson.Name,
		Description: fabricModJson.Description,
		McVersion:   version,
		Authors:     authors,
		Features:    features,
		Verified:    false,
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
		Custom: *customProperties,
	}

	return &addon, nil
}

func ParseRepos(repos map[string]bool) []*Addon {
	start := time.Now()

	total := len(repos)
	addons := make([]*Addon, 0, total)

	type Job struct {
		FullName string
		Verified bool
		Index    int
	}

	jobChan := make(chan Job)
	resultChan := make(chan *Addon)
	errorChan := make(chan error)
	var wg sync.WaitGroup

	workerCount := 10

	for i := range workerCount {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for job := range jobChan {
				addon, err := ParseRepo(job.FullName)
				if err != nil {
					errorChan <- fmt.Errorf("Failed parsing %s: %v", job.FullName, err)
					continue
				}
				addon.Verified = job.Verified
				resultChan <- addon
			}
		}(i)
	}

	go func() {
		idx := 1
		for fullName, verified := range repos {
			jobChan <- Job{
				FullName: fullName,
				Verified: verified,
				Index:    idx,
			}
			idx++
		}
		close(jobChan)
	}()

	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	for {
		select {
		case addon, ok := <-resultChan:
			if !ok {
				duration := time.Since(start)
				fmt.Printf(
					"Finished parsing in %s\n",
					duration.String(),
				)
				return addons
			}
			addons = append(addons, addon)

		case err, ok := <-errorChan:
			if !ok {
				continue
			}
			fmt.Printf("\t%s\n", err)
		}
	}
}
