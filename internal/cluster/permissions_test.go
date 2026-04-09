package cluster

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestFetchPermissions_Success(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "selfsubjectrulesreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectRulesReview{
			Status: authv1.SubjectRulesReviewStatus{
				ResourceRules: []authv1.ResourceRule{
					{Verbs: []string{"get", "list", "watch"}, Resources: []string{"pods"}},
					{Verbs: []string{"delete"}, Resources: []string{"deployments"}},
				},
			},
		}, nil
	})

	perms := FetchPermissions(context.Background(), client)

	testza.AssertFalse(t, perms.Inferred)
	testza.AssertLen(t, perms.Rules, 2)
}

func TestFetchPermissions_Error_SetsInferred(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "selfsubjectrulesreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewForbidden(schema.GroupResource{Resource: "selfsubjectrulesreviews"}, "", nil)
	})

	perms := FetchPermissions(context.Background(), client)

	testza.AssertTrue(t, perms.Inferred)
}

func TestCanMutate_ReadOnly_Rules(t *testing.T) {
	perms := PermissionSet{
		Rules: []authv1.ResourceRule{
			{Verbs: []string{"get", "list", "watch"}, Resources: []string{"pods"}},
		},
		Inferred: false,
	}
	testza.AssertFalse(t, perms.CanMutate())
}

func TestCanMutate_DeleteVerb(t *testing.T) {
	perms := PermissionSet{
		Rules: []authv1.ResourceRule{
			{Verbs: []string{"get", "delete"}, Resources: []string{"pods"}},
		},
		Inferred: false,
	}
	testza.AssertTrue(t, perms.CanMutate())
}

func TestCanMutate_Inferred_AlwaysTrue(t *testing.T) {
	perms := PermissionSet{Rules: nil, Inferred: true}
	testza.AssertTrue(t, perms.CanMutate())
}

func TestCanMutate_WildcardVerb(t *testing.T) {
	perms := PermissionSet{
		Rules: []authv1.ResourceRule{
			{Verbs: []string{"*"}, Resources: []string{"*"}},
		},
		Inferred: false,
	}
	testza.AssertTrue(t, perms.CanMutate())
}
