package scanner

import (
	"regexp"
	"strings"
)

type Config struct {
	BlacklistedRepos []string `json:"repo-blacklist"`
	BlacklistedDevs  []string `json:"developer-blacklist"`
	VerifiedAddons   struct {
		Verified            []string `json:"verified"`
		MinMinecraftVersion string   `json:"minimum_mc_version"`
		ValidateForks       bool     `json:"validate_forks"`
	} `json:"verified_addons"`
	ModuleDescriptions struct {
		Fetch        bool `json:"fetch"`
		OnlyVerified bool `json:"only_verified"`
		MinStarCount int  `json:"minimum_star_count"`
	} `json:"module_descriptions"`
	RequireMinecraftVersion bool `json:"require_mc_version"`
	IgnoreArchived          bool `json:"ignore_archived"`
	IgnoreForks             bool `json:"ignore_forks"`
	SuspicionTriggers       struct {
		NameLength        int `json:"name_len"`
		DescriptionLength int `json:"description_len"`
		FeatureCount      int `json:"feature_count"`
		SupportedVersions int `json:"supported_versions"`
	} `json:"suspicion_triggers"`
}

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
	Modules       []string `json:"modules"`
	Commands      []string `json:"commands"`
	HudElements   []string `json:"hud_elements"`
	CustomScreens []string `json:"custom_screens"`
	FeatureCount  int      `json:"feature_count"`
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
	Github        string   `json:"github"`
	Downloads     []string `json:"downloads"`
	LatestRelease string   `json:"latest_release"`
	Discord       string   `json:"discord"`
	Homepage      string   `json:"homepage"`
	Icon          string   `json:"icon"`
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
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Authors     any    `json:"authors"`
	Icon        string `json:"icon"`
	Entrypoints struct {
		Meteor any `json:"meteor"`
	} `json:"entrypoints"`
}

type release struct {
	Draft      bool `json:"draft"`
	Prerelease bool `json:"prerelease"`
	Assets     []struct {
		Name      string `json:"name"`
		Url       string `json:"browser_download_url"`
		Downloads int    `json:"download_count"`
	} `json:"assets"`
}

func validTag(tag string) (string, bool) {
	realTag, ok := validTags[strings.ToLower(tag)]
	return realTag.String(), ok
}
