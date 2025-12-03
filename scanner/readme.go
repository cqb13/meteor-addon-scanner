package scanner

import (
	"fmt"
	"os"
	"time"
)

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

func calculateInsights(addons []*Addon) EcosystemInsights {
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

// GenerateReadme creates a README.md file with statistics from the scan results
func GenerateReadme(result *ScanResult, outputPath string, executionTimeSeconds float64) error {
	// Calculate stats
	archivedCount := 0
	for _, addon := range result.Addons {
		if addon.Repo.Archived {
			archivedCount++
		}
	}

	validAddonsCount := len(result.Addons)
	invalidAddonsCount := len(result.InvalidAddons)
	totalScanned := validAddonsCount + invalidAddonsCount
	lastUpdated := time.Now().UTC().Format(time.RFC3339)

	// Calculate ecosystem insights
	insights := calculateInsights(result.Addons)

	// Format execution time
	var timeFormatted string
	if executionTimeSeconds >= 60 {
		minutes := int(executionTimeSeconds / 60)
		seconds := executionTimeSeconds - float64(minutes*60)
		timeFormatted = fmt.Sprintf("%dm %.2fs", minutes, seconds)
	} else {
		timeFormatted = fmt.Sprintf("%.2fs", executionTimeSeconds)
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
`,
		validAddonsCount,
		validAddonsCount,
		archivedCount,
		invalidAddonsCount,
		totalScanned,
		timeFormatted,
		lastUpdated,
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
