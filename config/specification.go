// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package config

import (
	"encoding/json"
	"time"
)

type Specification struct {
	AgentKey                  string        `json:"agentKey" split_words:"true" required:"true"`
	AgentPort                 int           `json:"agentPort" split_words:"true" default:"42899"`
	InitialDelay              time.Duration `json:"InitialDelay" split_words:"true" default:"5s"`
	NamespaceFilter           string        `json:"namespaceFilter" split_words:"true" required:"false"`
	LogKubernetesHttpRequests bool          `json:"LogKubernetesHttpRequests" split_words:"true" default:"false"`
	MatchLabels               Labels        `json:"matchLabels" split_words:"true" required:"false"`
	MatchLabelsExclude        Labels        `json:"matchLabelsExclude" split_words:"true" required:"false"`
	AgentRegistrationDebounce time.Duration `json:"agentRegistrationDebounce" split_words:"true" default:"1s"`
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
