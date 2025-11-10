package autoregistration

import (
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	extensionconfig "github.com/steadybit/extension-auto-registration-kubernetes/config"
)

func getCurrentRegistrations(httpClient *resty.Client) ([]ExtensionConfigAO, error) {
	var currentRegistrations *[]ExtensionConfigAO
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
	return []ExtensionConfigAO{}, nil
}

func removeMissingRegistrations(httpClient *resty.Client, currentRegistrations []ExtensionConfigAO, discoveredExtensions []ExtensionConfigAO) error {
	var combinedError error

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
				combinedError = errors.Join(combinedError, err)
			}
			if resp.IsError() {
				err := fmt.Errorf("failed to deregister extension: %v. Status: %s", currentRegistration, resp.Status())
				log.Error().Msg(err.Error())
				combinedError = errors.Join(combinedError, err)
			}
			if resp.IsSuccess() {
				log.Info().Msgf("De-Registered extension: %v", currentRegistration)
			}
		}
	}
	return combinedError
}

func addNewRegistrations(httpClient *resty.Client, currentRegistrations []ExtensionConfigAO, discoveredExtensions []ExtensionConfigAO) error {
	var combinedError error

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
				log.Error().Err(err).Msgf("Failed to register extension: %v", discoveredExtension)
				combinedError = errors.Join(combinedError, err)
			}
			if resp.IsError() {
				err := fmt.Errorf("failed to register extension: %v. Status: %s", discoveredExtension, resp.Status())
				log.Error().Msg(err.Error())
				combinedError = errors.Join(combinedError, err)
			}
			if resp.IsSuccess() {
				log.Info().Msgf("Registered extension: %v", discoveredExtension)
			}
		}
	}

	return combinedError
}

func extensionsEqual(a, b ExtensionConfigAO) bool {
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
