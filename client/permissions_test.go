package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestPermissionCheckResult_HasErrors(t *testing.T) {
	tests := []struct {
		name                  string
		permissionCheckResult *PermissionCheckResult
		expected              bool
	}{
		{
			name: "should return false when all permissions are OK",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get":      OK,
					"services/list": OK,
				},
			},
			expected: false,
		},
		{
			name: "should return true when some permissions have errors",
			permissionCheckResult: &PermissionCheckResult{
				Permissions: map[string]PermissionCheckOutcome{
					"pods/get":     OK,
					"services/get": ERROR,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permissionCheckResult.HasErrors()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckPermissions(t *testing.T) {
	tests := []struct {
		name             string
		setupReactions   func(*testclient.Clientset)
		expectedOutcomes map[string]PermissionCheckOutcome
	}{
		{
			name: "should return OK for allowed permissions",
			setupReactions: func(client *testclient.Clientset) {
				client.PrependReactor("create", "selfsubjectaccessreviews", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &authorizationv1.SelfSubjectAccessReview{
						Status: authorizationv1.SubjectAccessReviewStatus{
							Allowed: true,
						},
					}, nil
				})
			},
			expectedOutcomes: map[string]PermissionCheckOutcome{
				"services/get":   OK,
				"services/list":  OK,
				"services/watch": OK,
				"pods/get":       OK,
				"pods/list":      OK,
				"pods/watch":     OK,
			},
		},
		{
			name: "should return ERROR for denied permissions",
			setupReactions: func(client *testclient.Clientset) {
				client.PrependReactor("create", "selfsubjectaccessreviews", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					createAction := action.(ktesting.CreateAction)
					sar := createAction.GetObject().(*authorizationv1.SelfSubjectAccessReview)

					allowed := sar.Spec.ResourceAttributes.Resource == "services"

					return true, &authorizationv1.SelfSubjectAccessReview{
						Status: authorizationv1.SubjectAccessReviewStatus{
							Allowed: allowed,
						},
					}, nil
				})
			},
			expectedOutcomes: map[string]PermissionCheckOutcome{
				"services/get":   OK,
				"services/list":  OK,
				"services/watch": OK,
				"pods/get":       ERROR,
				"pods/list":      ERROR,
				"pods/watch":     ERROR,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := testclient.NewSimpleClientset()
			tt.setupReactions(fakeClient)

			result := checkPermissions(fakeClient)

			assert.NotNil(t, result)
			assert.Equal(t, len(tt.expectedOutcomes), len(result.Permissions))

			for permission, expectedOutcome := range tt.expectedOutcomes {
				actualOutcome, exists := result.Permissions[permission]
				assert.True(t, exists, "Expected permission %s to exist", permission)
				assert.Equal(t, expectedOutcome, actualOutcome, "Permission %s outcome mismatch", permission)
			}
		})
	}
}
