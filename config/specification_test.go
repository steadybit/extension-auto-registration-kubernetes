// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package config

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentApiCredentials(t *testing.T) {
	tests := []struct {
		name             string
		env              map[string]string
		expectedUsername string
		expectedPassword string
	}{
		{
			name:             "defaults to _ and the agent key",
			env:              map[string]string{"STEADYBIT_EXTENSION_AGENT_KEY": "agent-key"},
			expectedUsername: "_",
			expectedPassword: "agent-key",
		},
		{
			name: "uses the configured username and password",
			env: map[string]string{
				"STEADYBIT_EXTENSION_AGENT_KEY":          "agent-key",
				"STEADYBIT_EXTENSION_AGENT_API_USERNAME": "registrar",
				"STEADYBIT_EXTENSION_AGENT_API_PASSWORD": "api-secret",
			},
			expectedUsername: "registrar",
			expectedPassword: "api-secret",
		},
		{
			name:             "works without an agent key when a password is set",
			env:              map[string]string{"STEADYBIT_EXTENSION_AGENT_API_PASSWORD": "api-secret"},
			expectedUsername: "_",
			expectedPassword: "api-secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var spec Specification
			require.NoError(t, envconfig.Process("steadybit_extension", &spec))
			require.NoError(t, spec.Validate())

			username, password := spec.AgentApiCredentials()
			assert.Equal(t, tt.expectedUsername, username)
			assert.Equal(t, tt.expectedPassword, password)
		})
	}
}

func TestValidateRequiresAgentKeyOrPassword(t *testing.T) {
	var spec Specification
	require.NoError(t, envconfig.Process("steadybit_extension", &spec))
	assert.Error(t, spec.Validate())
}
