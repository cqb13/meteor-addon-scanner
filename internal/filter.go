package internal

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"strings"
)

// RemoveBlacklistedRepositories removes repos listed as blacklisted in the config
// Returns the number of repositories removed
func RemoveBlacklistedRepositories(config *scanner.Config, repos map[string]bool) int {
	removed := 0
	for _, repo := range config.BlacklistedRepos {
		lower := strings.ToLower(repo)

		for fullName := range repos {
			if strings.ToLower(fullName) == lower {
				delete(repos, fullName)
				removed++
				break
			}
		}
	}

	return removed
}

// RemoveBlacklistedDevelopers removes repos that belong to authors listed as blacklisted in the config
// Returns the number of repositories removed
func RemoveBlacklistedDevelopers(config *scanner.Config, repos map[string]bool) int {
	removed := 0
	for _, dev := range config.BlacklistedDevs {
		lower := strings.ToLower(dev)

		for fullName := range repos {
			if strings.HasPrefix(strings.ToLower(fullName), lower) {
				delete(repos, fullName)
				removed++
				break
			}
		}
	}

	return removed
}
