package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Stats struct {
	LastUpdated          string  `json:"last_updated"`
	ValidAddonsCount     int     `json:"valid_addons_count"`
	ArchivedAddonsCount  int     `json:"archived_addons_count"`
	InvalidAddonsCount   int     `json:"invalid_addons_count"`
	ExecutionTimeSeconds float64 `json:"execution_time_seconds"`
}

type AddonFeatures struct {
	Modules       []string `json:"modules"`
	Commands      []string `json:"commands"`
	HudElements   []string `json:"hud_elements"`
	CustomScreens []string `json:"custom_screens"`
	FeatureCount  int      `json:"feature_count"`
}

type AddonRepo struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	Fork         bool   `json:"fork"`
	Stars        int    `json:"stars"`
	Downloads    int    `json:"downloads"`
	LastUpdate   string `json:"last_update"`
	CreationDate string `json:"creation_date"`
}

type AddonLinks struct {
	Github        string   `json:"github"`
	Downloads     []string `json:"downloads"`
	LatestRelease string   `json:"latest_release"`
	Discord       string   `json:"discord"`
	Homepage      string   `json:"homepage"`
	Icon          string   `json:"icon"`
}

type Addon struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	McVersion   string        `json:"mc_version"`
	Authors     []string      `json:"authors"`
	Features    AddonFeatures `json:"features"`
	Verified    bool          `json:"verified"`
	Repo        AddonRepo     `json:"repo"`
	Links       AddonLinks    `json:"links"`
}

type AddonsData struct {
	Addons []Addon `json:"addons"`
}

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: go run generate-readme.go <stats.json> <addons.json> <output-readme.md>")
	}

	statsPath := os.Args[1]
	addonsPath := os.Args[2]
	outputPath := os.Args[3]

	if err := generateReadme(statsPath, addonsPath, outputPath); err != nil {
		log.Fatalf("Error generating README: %v", err)
	}

	fmt.Printf("Successfully generated README at %s\n", outputPath)
}

func generateReadme(statsPath, addonsPath, outputPath string) error {
	// Load stats
	statsFile, err := os.Open(statsPath)
	if err != nil {
		return fmt.Errorf("stats file not found at %s: %v", statsPath, err)
	}
	defer statsFile.Close()

	var stats Stats
	if err := json.NewDecoder(statsFile).Decode(&stats); err != nil {
		return fmt.Errorf("error decoding stats: %v", err)
	}

	// Load addons
	addonsFile, err := os.Open(addonsPath)
	if err != nil {
		return fmt.Errorf("addons file not found at %s: %v", addonsPath, err)
	}
	defer addonsFile.Close()

	var addonsData AddonsData
	if err := json.NewDecoder(addonsFile).Decode(&addonsData); err != nil {
		return fmt.Errorf("error decoding addons: %v", err)
	}

	// Calculate ecosystem insights
	insights := calculateInsights(addonsData.Addons)

	// Format execution time
	execTime := stats.ExecutionTimeSeconds
	var timeFormatted string
	if execTime >= 60 {
		minutes := int(execTime / 60)
		seconds := execTime - float64(minutes*60)
		timeFormatted = fmt.Sprintf("%dm %.2fs", minutes, seconds)
	} else {
		timeFormatted = fmt.Sprintf("%.2fs", execTime)
	}

	// Generate README content
	readme := fmt.Sprintf(`<div align="center">
<h1>Meteor Client Addons Database</h1>
</div>

<div align="center">
<p>
<img src="https://img.shields.io/badge/Go-1.22-blue?style=for-the-badge&logo=go&logoColor=white">
<img src="https://img.shields.io/badge/Updated-Daily_at_4pm_EST-green?style=for-the-badge">
<img src="https://img.shields.io/badge/License-MIT-blue?style=for-the-badge">
<img src="https://img.shields.io/badge/Total_Addons-%d-orange?style=for-the-badge">
</p>
</div>

<div align="center">
<h2>Scanner Statistics</h2>
</div>

<div align="center">

| Metric | Value |
|--------|-------|
| Valid Addons Found | %d |
| Archived Addons | %d |
| Invalid Addons Parsed | %d |
| Total Repositories Scanned | %d |
| Execution Time | %s |
| Last Updated | %s |

</div>

<div align="center">
<h2>Ecosystem Insights</h2>
</div>

<div align="center">

| Category | Count |
|----------|-------|
| Verified Addons | %d |
| Total Modules | %d |
| Total Commands | %d |
| Total HUD Elements | %d |
| Total Custom Screens | %d |
| Addons with Discord | %d |
| Addons with Releases | %d |
| Total Downloads | %s |

</div>

<div align="center">
<h2>Data Access</h2>
</div>

<div align="center">

Access the full database:

</div>

<div align="center">

https://raw.githubusercontent.com/cqb13/meteor-addon-scanner/addons/addons.json

</div>

<div align="center">

Access scanner statistics:

</div>

<div align="center">

https://raw.githubusercontent.com/cqb13/meteor-addon-scanner/addons/stats.json

</div>
`,
		stats.ValidAddonsCount,
		stats.ValidAddonsCount,
		stats.ArchivedAddonsCount,
		stats.InvalidAddonsCount,
		stats.ValidAddonsCount+stats.ArchivedAddonsCount+stats.InvalidAddonsCount,
		timeFormatted,
		stats.LastUpdated,
		insights.VerifiedCount,
		insights.TotalModules,
		insights.TotalCommands,
		insights.TotalHudElements,
		insights.TotalCustomScreens,
		insights.AddonsWithDiscord,
		insights.AddonsWithReleases,
		formatDownloads(insights.TotalDownloads),
	)

	// Write README
	if err := os.WriteFile(outputPath, []byte(readme), 0644); err != nil {
		return fmt.Errorf("error writing README: %v", err)
	}

	return nil
}

type EcosystemInsights struct {
	VerifiedCount      int
	TotalModules       int
	TotalCommands      int
	TotalHudElements   int
	TotalCustomScreens int
	AddonsWithDiscord  int
	AddonsWithReleases int
	TotalDownloads     int
}

func calculateInsights(addons []Addon) EcosystemInsights {
	insights := EcosystemInsights{}

	for _, addon := range addons {
		if addon.Verified {
			insights.VerifiedCount++
		}

		insights.TotalModules += len(addon.Features.Modules)
		insights.TotalCommands += len(addon.Features.Commands)
		insights.TotalHudElements += len(addon.Features.HudElements)
		insights.TotalCustomScreens += len(addon.Features.CustomScreens)

		if addon.Links.Discord != "" {
			insights.AddonsWithDiscord++
		}

		if len(addon.Links.Downloads) > 0 || addon.Links.LatestRelease != "" {
			insights.AddonsWithReleases++
		}

		insights.TotalDownloads += addon.Repo.Downloads
	}

	return insights
}

func formatDownloads(downloads int) string {
	if downloads >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(downloads)/1000000)
	} else if downloads >= 1000 {
		return fmt.Sprintf("%.1fK", float64(downloads)/1000)
	}
	return fmt.Sprintf("%d", downloads)
}
