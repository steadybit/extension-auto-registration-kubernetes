// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package config

import (
	"encoding/json"
	"errors"
	"time"
)

type Specification struct {
	// Deprecated: AgentKey is only read as a fallback for AgentApiPassword.
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

// ResolveAgentApiPassword falls back to the deprecated AgentKey when no AgentApiPassword is set, and reports whether
// the deprecated AgentKey was configured at all.
func (s *Specification) ResolveAgentApiPassword() (usesDeprecatedAgentKey bool, err error) {
	if s.AgentApiPassword == "" {
		s.AgentApiPassword = s.AgentKey
	}
	if s.AgentApiPassword == "" {
		return false, errors.New("STEADYBIT_EXTENSION_AGENT_API_PASSWORD must be set")
	}
	return s.AgentKey != "", nil
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
