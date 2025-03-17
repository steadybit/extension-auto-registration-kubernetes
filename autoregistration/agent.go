package autoregistration

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	extensionconfig "github.com/steadybit/extension-auto-registration-kubernetes/config"
)

func getCurrentRegistrations(httpClient *resty.Client) ([]extensionConfigAO, error) {
	var currentRegistrations *[]extensionConfigAO
	resp, err := httpClient.R().
		SetHeader("Accept", "application/json").
		SetResult(&currentRegistrations).
		Get("/extensions")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get extension registrations from the agent. Skip.")
		return nil, err
	}
	if resp.IsError() {
		log.Error().Msgf("Failed to get extension registrations from the agent: %s. Skip.", resp.Status())
		return nil, fmt.Errorf("failed to get extension registrations from the agent: %s", resp.Status())
	}
	if resp.IsSuccess() {
		if currentRegistrations != nil {
			log.Trace().Int("count", len(*currentRegistrations)).Msg("Got extension registrations from the agent")
			return *currentRegistrations, nil
		} else {
			log.Trace().Msg("No extension registrations found on the agent")
		}
	}
	return []extensionConfigAO{}, nil
}

func removeMissingRegistrations(httpClient *resty.Client, currentRegistrations []extensionConfigAO, discoveredExtensions []extensionConfigAO) {
	for _, currentRegistration := range currentRegistrations {
		found := false
		for _, discoveredExtension := range discoveredExtensions {
			if extensionsEqual(currentRegistration, discoveredExtension) {
				found = true
				break
			}
		}
		if !found {
			resp, err := httpClient.R().
				SetHeader("Content-Type", "application/json").
				SetBasicAuth("_", extensionconfig.Config.AgentKey).
				SetBody(currentRegistration).
				Delete("/extensions")
			if err != nil {
				log.Error().Err(err).Msgf("Failed to deregister extension: %v", currentRegistration)
			}
			if resp.IsError() {
				log.Error().Msgf("Failed to deregister extension: %v. Status: %s", currentRegistration, resp.Status())
			}
			if resp.IsSuccess() {
				log.Info().Msgf("De-Registered extension: %v", currentRegistration)
			}
		}
	}
}

func addNewRegistrations(httpClient *resty.Client, currentRegistrations []extensionConfigAO, discoveredExtensions []extensionConfigAO) {
	for _, discoveredExtension := range discoveredExtensions {
		found := false
		for _, currentRegistration := range currentRegistrations {
			if extensionsEqual(currentRegistration, discoveredExtension) {
				found = true
				break
			}
		}
		if !found {
			resp, err := httpClient.R().
				SetHeader("Content-Type", "application/json").
				SetBasicAuth("_", extensionconfig.Config.AgentKey).
				SetBody(discoveredExtension).
				Post("/extensions")
			if err != nil {
				log.Error().Err(err).Msgf("Failed to registern extension: %v", discoveredExtension)
			}
			if resp.IsError() {
				log.Error().Msgf("Failed to register extension: %v. Status: %s", discoveredExtension, resp.Status())
			}
			if resp.IsSuccess() {
				log.Info().Msgf("Registered extension: %v", discoveredExtension)
			}
		}
	}
}

func extensionsEqual(a, b extensionConfigAO) bool {
	if a.Url != b.Url {
		return false
	}
	if !compareRestrictedPorts(a.RestrictedPorts, b.RestrictedPorts) {
		return false
	}
	if !compareRestrictedIps(a.RestrictedIps, b.RestrictedIps) {
		return false
	}
	return true
}

func compareRestrictedPorts(a, b map[int]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, valA := range a {
		if valB, exists := b[key]; !exists || valA != valB {
			return false
		}
	}
	return true
}

func compareRestrictedIps(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ipMap := make(map[string]struct{}, len(a))
	for _, ip := range a {
		ipMap[ip] = struct{}{}
	}
	for _, ip := range b {
		if _, exists := ipMap[ip]; !exists {
			return false
		}
	}
	return true
}
