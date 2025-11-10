package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredPermission_Key(t *testing.T) {
	tests := []struct {
		name               string
		requiredPermission requiredPermission
		verb               string
		expected           string
	}{
		{
			name: "key with group and resource",
			requiredPermission: requiredPermission{
				group:    "apps",
				resource: "deployments",
			},
			verb:     "get",
			expected: "apps/deployments/get",
		},
		{
			name: "key without group (core API)",
			requiredPermission: requiredPermission{
				group:    "",
				resource: "pods",
			},
			verb:     "list",
			expected: "pods/list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.requiredPermission.Key(tt.verb)
			assert.Equal(t, tt.expected, result)
		})
	}
}
