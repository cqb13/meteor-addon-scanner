package internal

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"strings"
)

// RemoveBlacklistedRepositories removes repos listed as blacklisted in the config
// Returns the number of repositories removed
func RemoveBlacklistedRepositories(config *scanner.Config, repos map[string]struct{}) int {
	blacklist := make(map[string]struct{}, len(config.BlacklistedRepos))
	for _, repo := range config.BlacklistedRepos {
		blacklist[strings.ToLower(repo)] = struct{}{}
	}

	removed := 0
	for fullName := range repos {
		if _, exist := blacklist[strings.ToLower(fullName)]; exist {
			delete(repos, fullName)
			removed++
		}
	}

	return removed
}

// RemoveBlacklistedDevelopers removes repos that belong to authors listed as blacklisted in the config
// Returns the number of repositories removed
func RemoveBlacklistedDevelopers(config *scanner.Config, repos map[string]struct{}) int {
	blacklist := make(map[string]struct{}, len(config.BlacklistedDevs))
	for _, dev := range config.BlacklistedDevs {
		blacklist[strings.ToLower(dev)] = struct{}{}
	}

	removed := 0
	for fullName := range repos {
		owner, _, ok := strings.Cut(fullName, "/")
		if !ok {
			continue
		}
		if _, bad := blacklist[strings.ToLower(owner)]; bad {
			delete(repos, fullName)
			removed++
		}
	}

	return removed
}
