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

func TestPermissionCheckResult_hasPermissions(t *testing.T) {
	tests := []struct {
		name                 string
		permissionCheckResult *PermissionCheckResult
		requiredPermissions   []string
		expected             bool
	}{
		{
			name: "should return true when all permissions are granted",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get":      OK,
					"pods/list":     OK,
					"services/get":  OK,
					"services/list": OK,
				},
			},
			requiredPermissions: []string{"pods/get", "services/list"},
			expected:            true,
		},
		{
			name: "should return false when some permissions are missing",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get":     OK,
					"services/get": ERROR,
				},
			},
			requiredPermissions: []string{"pods/get", "services/get"},
			expected:            false,
		},
		{
			name: "should return false when permission is not in map",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get": OK,
				},
			},
			requiredPermissions: []string{"pods/get", "pods/list"},
			expected:            false,
		},
		{
			name: "should return true when no permissions required",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get": OK,
				},
			},
			requiredPermissions: []string{},
			expected:            true,
		},
		{
			name: "should return false when permission has error outcome",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get":  OK,
					"pods/list": ERROR,
				},
			},
			requiredPermissions: []string{"pods/get", "pods/list"},
			expected:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permissionCheckResult.hasPermissions(tt.requiredPermissions)
			assert.Equal(t, tt.expected, result)
		})
	}
}
