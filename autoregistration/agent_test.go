// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package autoregistration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	extensionconfig "github.com/steadybit/extension-auto-registration-kubernetes/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        ExtensionConfigAO
		b        ExtensionConfigAO
		expected bool
	}{
		{
			name: "identical extensions should be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with different URLs should not be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8081",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different restricted ports should not be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8081: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different port descriptions should not be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ServicePort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different restricted IPs should not be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.2"},
			},
			expected: false,
		},
		{
			name: "extensions with same IPs in different order should be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1", "192.168.1.2"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.2", "192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with empty restricted ports should be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with empty restricted IPs should be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{},
			},
			expected: true,
		},
		{
			name: "extensions with nil vs empty restricted ports should be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: nil,
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with nil vs empty restricted IPs should be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   nil,
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{},
			},
			expected: true,
		},
		{
			name: "extensions with multiple ports should be equal",
			a: ExtensionConfigAO{
				Url: "http://test.example.com:8080",
				RestrictedPorts: map[int]string{
					8080: "ContainerPort",
					8081: "LivenessProbe",
					8082: "ReadinessProbe",
				},
				RestrictedIps: []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url: "http://test.example.com:8080",
				RestrictedPorts: map[int]string{
					8080: "ContainerPort",
					8081: "LivenessProbe",
					8082: "ReadinessProbe",
				},
				RestrictedIps: []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with different number of ports should not be equal",
			a: ExtensionConfigAO{
				Url: "http://test.example.com:8080",
				RestrictedPorts: map[int]string{
					8080: "ContainerPort",
					8081: "LivenessProbe",
				},
				RestrictedIps: []string{"192.168.1.1"},
			},
			b: ExtensionConfigAO{
				Url: "http://test.example.com:8080",
				RestrictedPorts: map[int]string{
					8080: "ContainerPort",
				},
				RestrictedIps: []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different number of IPs should not be equal",
			a: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1", "192.168.1.2"},
			},
			b: ExtensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extensionsEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveMissingRegistrationsKeepsStillDiscoveredUrls(t *testing.T) {
	var deleted []string
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		deleted = append(deleted, string(body))
	}))
	defer agent.Close()

	err := removeMissingRegistrations(resty.New().SetBaseURL(agent.URL),
		[]ExtensionConfigAO{
			{Url: "http://changed:8080", RestrictedIps: []string{"192.168.1.1"}},
			{Url: "http://gone:8080", RestrictedIps: []string{"192.168.1.2"}},
		},
		[]ExtensionConfigAO{
			{Url: "http://changed:8080", RestrictedIps: []string{"192.168.1.1", "192.168.1.3"}},
		})

	require.NoError(t, err)
	assert.Equal(t, []string{`{"url":"http://gone:8080","restrictedIps":["192.168.1.2"]}`}, deleted)
}

func TestRegistrationCallsSendConfiguredCredentials(t *testing.T) {
	original := extensionconfig.Config
	t.Cleanup(func() { extensionconfig.Config = original })
	extensionconfig.Config = extensionconfig.Specification{AgentApiUsername: "registrar", AgentApiPassword: "api-secret"}

	type call struct{ method, username, password string }
	var calls []call
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, _ := r.BasicAuth()
		calls = append(calls, call{r.Method, username, password})
	}))
	defer agent.Close()
	httpClient := resty.New().SetBaseURL(agent.URL)

	require.NoError(t, addNewRegistrations(httpClient, nil, []ExtensionConfigAO{{Url: "http://new:8080"}}))
	require.NoError(t, removeMissingRegistrations(httpClient, []ExtensionConfigAO{{Url: "http://gone:8080"}}, nil))

	assert.Equal(t, []call{
		{http.MethodPost, "registrar", "api-secret"},
		{http.MethodDelete, "registrar", "api-secret"},
	}, calls)
}
