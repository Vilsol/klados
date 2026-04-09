package cluster

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func fakeConnWith(client *fake.Clientset) *Connection {
	return &Connection{Clientset: client}
}

func TestCheckHealth_NodeForbidden_SetsPermissionDenied(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewForbidden(schema.GroupResource{Resource: "nodes"}, "", nil)
	})
	// componentstatuses → 404 (not found) so components treated as unknown
	client.PrependReactor("list", "componentstatuses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewNotFound(schema.GroupResource{Resource: "componentstatuses"}, "")
	})

	h := CheckHealth(context.Background(), fakeConnWith(client))

	testza.AssertTrue(t, h.Nodes.PermissionDenied)
	testza.AssertEqual(t, 0, h.Nodes.Total)
}

func TestCheckHealth_ComponentStatuses404_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "componentstatuses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewNotFound(schema.GroupResource{Resource: "componentstatuses"}, "")
	})

	h := CheckHealth(context.Background(), fakeConnWith(client))

	testza.AssertLen(t, h.Components, 0)
}

func TestCheckHealth_EmptyComponentStatuses_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "componentstatuses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &corev1.ComponentStatusList{
			TypeMeta: metav1.TypeMeta{Kind: "ComponentStatusList"},
			Items:    []corev1.ComponentStatus{},
		}, nil
	})

	h := CheckHealth(context.Background(), fakeConnWith(client))

	testza.AssertLen(t, h.Components, 0)
}
