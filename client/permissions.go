package client

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-auto-registration-kubernetes/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)
import authorizationv1 "k8s.io/api/authorization/v1"

type PermissionCheckResult struct {
	Permissions map[string]PermissionCheckOutcome
}

type PermissionCheckOutcome string

const (
	ERROR PermissionCheckOutcome = "error"
	OK    PermissionCheckOutcome = "ok"
)

type requiredPermission struct {
	verbs    []string
	group    string
	resource string
}

func (p *requiredPermission) Key(verb string) string {
	result := ""
	if p.group != "" {
		result = p.group + "/"
	}
	result = result + p.resource + "/"
	result = result + verb
	return result
}

var requiredPermissions = []requiredPermission{
	{group: "", resource: "services", verbs: []string{"get", "list", "watch"}},
	{group: "", resource: "pods", verbs: []string{"get", "list", "watch"}},
}

func checkPermissions(client *kubernetes.Clientset) *PermissionCheckResult {
	result := make(map[string]PermissionCheckOutcome)
	reviews := client.AuthorizationV1().SelfSubjectAccessReviews()
	errors := false

	for _, p := range requiredPermissions {
		for _, verb := range p.verbs {
			sar := authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace: config.Config.NamespaceFilter,
						Verb:      verb,
						Resource:  p.resource,
						Group:     p.group,
					},
				},
			}
			review, err := reviews.Create(context.TODO(), &sar, metav1.CreateOptions{})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check permission %s", p.Key(verb))
			}
			if err != nil || !review.Status.Allowed {
				result[p.Key(verb)] = ERROR
				errors = true
			} else {
				result[p.Key(verb)] = OK
			}
		}
	}

	logPermissionCheckResult(result)
	if errors {
		log.Fatal().Msg("Required permissions are missing. Exit now.")
	}

	return &PermissionCheckResult{
		Permissions: result,
	}
}

func logPermissionCheckResult(permissions map[string]PermissionCheckOutcome) {
	log.Info().Msg("Permission check results:")
	allGood := true
	for k, v := range permissions {
		if v == OK {
			log.Debug().Str("permission", k).Str("result", string(v)).Msg("Permission granted.")
		} else if v == ERROR {
			log.Error().Str("permission", k).Str("result", string(v)).Msg("Permission missing.")
			allGood = false
		}
	}
	if allGood {
		log.Info().Msg("All permissions granted.")
	}
}

func (p *PermissionCheckResult) hasPermissions(requiredPermissions []string) bool {
	for _, rp := range requiredPermissions {
		outcome, ok := p.Permissions[rp]
		if !ok || outcome != OK {
			return false
		}
	}
	return true
}
