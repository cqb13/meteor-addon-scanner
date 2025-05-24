package main

import (
	"dev/cqb13/meteor-addon-scanner/config"
	"dev/cqb13/meteor-addon-scanner/scanner"
	"fmt"
	"os"
	"regexp"

	"github.com/joho/godotenv"
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

var conf config.Config

func locator() []string {
	// matches patterns like `add(new SomeFeatureName(...))`
	// and captures the feature name (e.g., "SomeFeatureName")
	var FEATURE_RE = regexp.MustCompile(`(?:add\(new )([^(]+)(?:\([^)]*)\)\)`)

	// matches Discord invite links, supporting various domains
	// and formats (e.g., "https://discord.gg/abc123", "discord.com/invite/abc")
	var INVITE_RE = regexp.MustCompile(`((?:https?:\/\/)?(?:www\.)?(?:discord\.(?:gg|io|me|li|com)|discordapp\.com/invite|dsc\.gg)/[a-zA-Z0-9\-\/]+)`)

	// matches Maven-style Minecraft version identifiers like
	// 'com.mojang:minecraft:1.20.4' and captures the version part (e.g., "1.20.4")
	var MCVER_RE = regexp.MustCompile(`(?:['"]com\.mojang:minecraft:)([0-9a-z.]+)(?:['"])`)

	_, _, _ = FEATURE_RE, INVITE_RE, MCVER_RE

	var repos []string

	return repos
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load env file: ", err)
		os.Exit(1)
	}

	var key string = os.Getenv("KEY")
	scanner.InitDefaultHeaders(key)

	conf = config.ParseConfig()
	scanner.SetConfig(conf)
	fmt.Println("Locating Repositories")
	scanner.SleepIfRateLimited(scanner.Core)
}
