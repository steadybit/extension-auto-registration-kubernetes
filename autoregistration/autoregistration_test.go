// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package autoregistration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestClusterIPsOfService(t *testing.T) {
	tests := []struct {
		name     string
		spec     corev1.ServiceSpec
		expected []string
	}{
		{
			name:     "single cluster IP",
			spec:     corev1.ServiceSpec{ClusterIP: "172.20.14.11"},
			expected: []string{"172.20.14.11"},
		},
		{
			name:     "dual-stack cluster IPs without duplicates",
			spec:     corev1.ServiceSpec{ClusterIP: "172.20.14.11", ClusterIPs: []string{"172.20.14.11", "fd00::42"}},
			expected: []string{"172.20.14.11", "fd00::42"},
		},
		{
			name:     "headless service",
			spec:     corev1.ServiceSpec{ClusterIP: "None", ClusterIPs: []string{"None"}},
			expected: []string{},
		},
		{
			name:     "no cluster IP",
			spec:     corev1.ServiceSpec{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, clusterIPsOfService(&corev1.Service{Spec: tt.spec}))
		})
	}
}

func TestMergeByUrl(t *testing.T) {
	serviceUrl := "http://test-service.default.svc.cluster.local:8085"
	merged := mergeByUrl(map[string][]ExtensionConfigAO{
		"default/test-pod-2": {{
			Url:             serviceUrl,
			RestrictedPorts: map[int]string{8085: "ServicePort", 8082: "ReadinessProbe"},
			RestrictedIps:   []string{"172.20.14.11", "192.168.1.2"},
		}},
		"default/test-pod": {{
			Url:             serviceUrl,
			RestrictedPorts: map[int]string{8085: "ServicePort", 8081: "LivenessProbe"},
			RestrictedIps:   []string{"172.20.14.11", "192.168.1.1"},
		}},
		"default/daemonset-pod": {{
			Url:           "http://192.168.1.3:8080",
			RestrictedIps: []string{"192.168.1.3"},
		}},
	})

	assert.Equal(t, []ExtensionConfigAO{
		{
			Url:             "http://192.168.1.3:8080",
			RestrictedPorts: map[int]string{},
			RestrictedIps:   []string{"192.168.1.3"},
		},
		{
			Url:             serviceUrl,
			RestrictedPorts: map[int]string{8085: "ServicePort", 8081: "LivenessProbe", 8082: "ReadinessProbe"},
			RestrictedIps:   []string{"172.20.14.11", "192.168.1.1", "192.168.1.2"},
		},
	}, merged)
}

func TestMergeByUrlDoesNotModifyPodConfigs(t *testing.T) {
	serviceUrl := "http://test-service.default.svc.cluster.local:8085"
	first := ExtensionConfigAO{Url: serviceUrl, RestrictedPorts: map[int]string{8085: "ServicePort"}, RestrictedIps: []string{"192.168.1.1"}}
	second := ExtensionConfigAO{Url: serviceUrl, RestrictedPorts: map[int]string{8081: "LivenessProbe"}, RestrictedIps: []string{"192.168.1.2"}}

	mergeByUrl(map[string][]ExtensionConfigAO{"default/a": {first}, "default/b": {second}})

	assert.Equal(t, map[int]string{8085: "ServicePort"}, first.RestrictedPorts)
	assert.Equal(t, []string{"192.168.1.1"}, first.RestrictedIps)
}
