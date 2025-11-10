package autoregistration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtensionsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        extensionConfigAO
		b        extensionConfigAO
		expected bool
	}{
		{
			name: "identical extensions should be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with different URLs should not be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8081",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different restricted ports should not be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8081: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different port descriptions should not be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ServicePort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: false,
		},
		{
			name: "extensions with different restricted IPs should not be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.2"},
			},
			expected: false,
		},
		{
			name: "extensions with same IPs in different order should be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1", "192.168.1.2"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.2", "192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with empty restricted ports should be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with empty restricted IPs should be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{},
			},
			expected: true,
		},
		{
			name: "extensions with nil vs empty restricted ports should be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: nil,
				RestrictedIps:   []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{},
				RestrictedIps:   []string{"192.168.1.1"},
			},
			expected: true,
		},
		{
			name: "extensions with nil vs empty restricted IPs should be equal",
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   nil,
			},
			b: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{},
			},
			expected: true,
		},
		{
			name: "extensions with multiple ports should be equal",
			a: extensionConfigAO{
				Url: "http://test.example.com:8080",
				RestrictedPorts: map[int]string{
					8080: "ContainerPort",
					8081: "LivenessProbe",
					8082: "ReadinessProbe",
				},
				RestrictedIps: []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
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
			a: extensionConfigAO{
				Url: "http://test.example.com:8080",
				RestrictedPorts: map[int]string{
					8080: "ContainerPort",
					8081: "LivenessProbe",
				},
				RestrictedIps: []string{"192.168.1.1"},
			},
			b: extensionConfigAO{
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
			a: extensionConfigAO{
				Url:             "http://test.example.com:8080",
				RestrictedPorts: map[int]string{8080: "ContainerPort"},
				RestrictedIps:   []string{"192.168.1.1", "192.168.1.2"},
			},
			b: extensionConfigAO{
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
