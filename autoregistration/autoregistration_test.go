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
