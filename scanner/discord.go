package scanner

import (
	"fmt"
	"regexp"
)

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
