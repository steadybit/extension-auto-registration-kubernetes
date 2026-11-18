// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package config

import (
	"encoding/json"
	"errors"
	"time"
)

type Specification struct {
	AgentKey                            string        `json:"agentKey" split_words:"true" required:"false"`
	AgentApiUsername                    string        `json:"agentApiUsername" split_words:"true" default:"_"`
	AgentApiPassword                    string        `json:"agentApiPassword" split_words:"true" required:"false"`
	AgentPort                           int           `json:"agentPort" split_words:"true" default:"42899"`
	NamespaceFilter                     string        `json:"namespaceFilter" split_words:"true" required:"false"`
	LogKubernetesHttpRequests           bool          `json:"LogKubernetesHttpRequests" split_words:"true" default:"false"`
	MatchLabels                         Labels        `json:"matchLabels" split_words:"true" required:"false"`
	MatchLabelsExclude                  Labels        `json:"matchLabelsExclude" split_words:"true" required:"false"`
	AgentRegistrationInitialDelay       time.Duration `json:"agentRegistrationInitialDelay" split_words:"true" default:"25s"`
	AgentRegistrationInterval           time.Duration `json:"agentRegistrationInterval" split_words:"true" default:"1s"`
	AgentRegistrationIntervalAfterError time.Duration `json:"agentRegistrationIntervalAfterError" split_words:"true" default:"5s"`
}

// AgentApiCredentials returns the Basic-auth credentials for the agent API. Like the agent itself, the password
// falls back to the agent key when no dedicated API password is configured.
func (s *Specification) AgentApiCredentials() (string, string) {
	if s.AgentApiPassword != "" {
		return s.AgentApiUsername, s.AgentApiPassword
	}
	return s.AgentApiUsername, s.AgentKey
}

func (s *Specification) Validate() error {
	if s.AgentKey == "" && s.AgentApiPassword == "" {
		return errors.New("either STEADYBIT_EXTENSION_AGENT_KEY or STEADYBIT_EXTENSION_AGENT_API_PASSWORD must be set")
	}
	return nil
}

type Labels []Label
type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (j *Labels) UnmarshalText(text []byte) error {
	if len(text) == 0 || string(text) == "[]" {
		*j = Labels{}
		return nil
	}
	return json.Unmarshal(text, (*[]Label)(j))
}
