// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package config

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAgentApiPassword(t *testing.T) {
	tests := []struct {
		name                   string
		env                    map[string]string
		expectedUsername       string
		expectedPassword       string
		usesDeprecatedAgentKey bool
	}{
		{
			name:             "uses the configured username and password",
			env:              map[string]string{"STEADYBIT_EXTENSION_AGENT_API_USERNAME": "registrar", "STEADYBIT_EXTENSION_AGENT_API_PASSWORD": "api-secret"},
			expectedUsername: "registrar",
			expectedPassword: "api-secret",
		},
		{
			name:                   "falls back to the deprecated agent key",
			env:                    map[string]string{"STEADYBIT_EXTENSION_AGENT_KEY": "agent-key"},
			expectedUsername:       "_",
			expectedPassword:       "agent-key",
			usesDeprecatedAgentKey: true,
		},
		{
			name:                   "prefers the password over the deprecated agent key",
			env:                    map[string]string{"STEADYBIT_EXTENSION_AGENT_KEY": "agent-key", "STEADYBIT_EXTENSION_AGENT_API_PASSWORD": "api-secret"},
			expectedUsername:       "_",
			expectedPassword:       "api-secret",
			usesDeprecatedAgentKey: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var spec Specification
			require.NoError(t, envconfig.Process("steadybit_extension", &spec))

			usesDeprecatedAgentKey, err := spec.ResolveAgentApiPassword()

			require.NoError(t, err)
			assert.Equal(t, tt.usesDeprecatedAgentKey, usesDeprecatedAgentKey)
			assert.Equal(t, tt.expectedUsername, spec.AgentApiUsername)
			assert.Equal(t, tt.expectedPassword, spec.AgentApiPassword)
		})
	}
}

func TestResolveAgentApiPasswordRequiresAPassword(t *testing.T) {
	var spec Specification
	require.NoError(t, envconfig.Process("steadybit_extension", &spec))

	_, err := spec.ResolveAgentApiPassword()

	assert.Error(t, err)
}
