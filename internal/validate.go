package internal

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"encoding/json"
	"fmt"
	"time"
)

type ForkValidationResult int

const (
	valid ForkValidationResult = iota
	invalidChildTooOld
	invalidParentTooRecent
)

type forkedRepository struct {
	Parent struct {
		PushedAt string `json:"pushed_at"`
	} `json:"parent"`
}

func ValidateForkedVerifiedAddons(addons []*scanner.Addon) map[string]string {
	log := make(map[string]string)

	for _, addon := range addons {
		if !addon.Verified || !addon.Repo.Fork {
			continue
		}

		result, err := checkParentAndChildUpdateDates(*addon)
		if err != nil {
			log[addon.Repo.Id] = fmt.Sprintf("Failed to check, %s", err)
			continue
		}

		switch result {
		case valid:
			log[addon.Repo.Id] = "Valid"
			continue
		case invalidChildTooOld:
			log[addon.Repo.Id] = "Repo has not been updated in 6 months -> no longer verified"
			addon.Verified = false
			continue
		case invalidParentTooRecent:
			log[addon.Repo.Id] = "Parent repo was updated within 6 months of the fork -> no longer verified"
			addon.Verified = false
			continue
		}
	}

	return log
}

func checkParentAndChildUpdateDates(addon scanner.Addon) (ForkValidationResult, error) {
	if !addon.Repo.Fork {
		return 0, fmt.Errorf("%s is not a fork", addon.Repo.Id)
	}
	currentTime := time.Now()

	childUpdateTime, err := time.Parse(time.RFC3339, addon.Repo.LastUpdate)
	if err != nil {
		return 0, err
	}

	// Reject if the child hasn't been updated in 6 months
	if childUpdateTime.AddDate(0, 6, 0).Before(currentTime) {
		return invalidChildTooOld, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s", addon.Repo.Id)
	bytes, err := scanner.MakeGetRequest(url)
	if err != nil {
		return 0, err
	}

	var repo forkedRepository
	err = json.Unmarshal(bytes, &repo)
	if err != nil {
		return 0, err
	}

	parentUpdateTime, err := time.Parse(time.RFC3339, repo.Parent.PushedAt)
	if err != nil {
		return 0, err
	}

	// Reject if the parent has been updated within 6 months of now
	if parentUpdateTime.AddDate(0, 6, 0).After(currentTime) {
		return invalidParentTooRecent, nil
	}

	return valid, nil
}

func DetectSuspiciousAddons(addons []*scanner.Addon, config *scanner.Config) map[string][]string {
	suspicious := make(map[string][]string)
	for _, addon := range addons {
		reasons := make([]string, 0)

		if len(addon.Name) >= config.SuspicionTriggers.NameLength {
			reasons = append(reasons, fmt.Sprintf("[Exceeding name length (%d)]", len(addon.Name)))
		}

		if len(addon.Description) >= config.SuspicionTriggers.DescriptionLength {
			reasons = append(reasons, fmt.Sprintf("[Exceeding github description length (%d)]", len(addon.Description)))
		}

		if len(addon.Custom.Description) >= config.SuspicionTriggers.DescriptionLength {
			reasons = append(reasons, fmt.Sprintf("[Exceeding custom description length (%d)]", len(addon.Custom.Description)))
		}

		if addon.Features.FeatureCount >= config.SuspicionTriggers.FeatureCount {
			reasons = append(reasons, fmt.Sprintf("[Exceeding feature limit (%d)]", addon.Features.FeatureCount))
		}

		if len(addon.Custom.SupportedVersions) >= config.SuspicionTriggers.SupportedVersions {
			reasons = append(reasons, fmt.Sprintf("[Exceeding supported version limit (%d)]", len(addon.Custom.SupportedVersions)))
		}

		if len(reasons) > 0 {
			suspicious[addon.Repo.Id] = reasons
		}
	}

	return suspicious
}
