package cluster

import (
	"context"

	"github.com/Vilsol/slox"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PermissionSet holds the result of a SelfSubjectRulesReview check.
// SelfSubjectRulesReview is namespace-scoped (checked against kube-system),
// so this is a coarse read/write signal, not fine-grained per-resource auth.
type PermissionSet struct {
	Rules    []authv1.ResourceRule `json:"rules"`
	Inferred bool                  `json:"inferred"` // true = RulesReview failed; assume full access
}

// CanMutate returns true if the permission set allows any write verb.
// When Inferred is true (fallback), full access is assumed.
func (p PermissionSet) CanMutate() bool {
	if p.Inferred {
		return true
	}
	for _, rule := range p.Rules {
		for _, verb := range rule.Verbs {
			switch verb {
			case "*", "delete", "patch", "update", "create":
				return true
			}
		}
	}
	return false
}

// FetchPermissions calls SelfSubjectRulesReview in the kube-system namespace.
// Any error (including 403 or unsupported API) results in Inferred: true so the
// UI defaults to full access rather than silently locking the user out.
func FetchPermissions(ctx context.Context, client kubernetes.Interface) PermissionSet {
	review := &authv1.SelfSubjectRulesReview{
		Spec: authv1.SelfSubjectRulesReviewSpec{
			Namespace: "kube-system",
		},
	}

	result, err := client.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		slox.Warn(ctx, "SelfSubjectRulesReview failed, assuming full access", "error", err)
		return PermissionSet{Inferred: true}
	}

	return PermissionSet{
		Rules:    result.Status.ResourceRules,
		Inferred: false,
	}
}
