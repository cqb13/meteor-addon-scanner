package scanner

import (
	"regexp"
)

type Addon struct {
	Name         string
	Description  string
	McVersion    string
	Authors      []string
	Features     []string
	FeatureCount int
	Verified     bool
	Repo         Repo
	Links        Links
}

type Repo struct {
	Id         string
	Owner      string
	Name       string
	Archived   bool
	Fork       bool
	Stars      int
	Downloads  int
	LastUpdate string
}

type Links struct {
	Github   string
	Download string
	Discord  string
	Homepage string
}

// matches patterns like `add(new SomeFeatureName(...))`
// and captures the feature name (e.g., "SomeFeatureName")
var FEATURE_RE = regexp.MustCompile(`(?:add\(new )([^(]+)(?:\([^)]*)\)\)`)

// matches Discord invite links, supporting various domains
// and formats (e.g., "https://discord.gg/abc123", "discord.com/invite/abc")
var INVITE_RE = regexp.MustCompile(`((?:https?:\/\/)?(?:www\.)?(?:discord\.(?:gg|io|me|li|com)|discordapp\.com/invite|dsc\.gg)/[a-zA-Z0-9\-\/]+)`)

// matches Maven-style Minecraft version identifiers like
// 'com.mojang:minecraft:1.20.4' and captures the version part (e.g., "1.20.4")
var MCVER_RE = regexp.MustCompile(`(?:['"]com\.mojang:minecraft:)([0-9a-z.]+)(?:['"])`)

func ParseRepos(repos []string) {
	for i, repo := range repos {
		_, _ = i, repo
	}
}
