package resource

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/Vilsol/klados/internal/cluster"
)

type ConnectionProvider interface {
	GetConnection(contextName string) (*cluster.Connection, error)
}

type ResourceEngine struct {
	clusterMgr  ConnectionProvider
	enricherReg *EnricherRegistry
}

func NewResourceEngine(mgr ConnectionProvider, enricherReg *EnricherRegistry) *ResourceEngine {
	return &ResourceEngine{
		clusterMgr:  mgr,
		enricherReg: enricherReg,
	}
}

func ParseGVR(gvr string) (schema.GroupVersionResource, error) {
	lastDot := strings.LastIndex(gvr, ".")
	if lastDot == -1 {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid GVR %q", gvr)
	}
	resource := gvr[lastDot+1:]
	rest := gvr[:lastDot]

	secondLastDot := strings.LastIndex(rest, ".")
	if secondLastDot == -1 {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid GVR %q", gvr)
	}
	version := rest[secondLastDot+1:]
	group := rest[:secondLastDot]

	if group == "core" {
		group = ""
	}

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}, nil
}

func (e *ResourceEngine) client(contextName, gvr, namespace string) (dynamic.ResourceInterface, error) {
	conn, err := e.clusterMgr.GetConnection(contextName)
	if err != nil {
		return nil, err
	}

	parsed, err := ParseGVR(gvr)
	if err != nil {
		return nil, err
	}

	ri := conn.Dynamic.Resource(parsed)
	if namespace != "" {
		return ri.Namespace(namespace), nil
	}
	return ri, nil
}

func (e *ResourceEngine) enrich(gvr string, item *unstructured.Unstructured) {
	for _, enricher := range e.enricherReg.GetAll(gvr) {
		_ = enricher.Enrich(item)
	}
}

func (e *ResourceEngine) List(ctx context.Context, contextName, gvr, namespace string) ([]map[string]any, error) {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return nil, err
	}

	list, err := client.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", gvr, err)
	}

	result := make([]map[string]any, len(list.Items))
	for i := range list.Items {
		e.enrich(gvr, &list.Items[i])
		result[i] = list.Items[i].Object
	}
	return result, nil
}

func (e *ResourceEngine) Get(ctx context.Context, contextName, gvr, namespace, name string) (map[string]any, error) {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return nil, err
	}

	obj, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting %s/%s: %w", gvr, name, err)
	}

	e.enrich(gvr, obj)
	return obj.Object, nil
}

func (e *ResourceEngine) Delete(ctx context.Context, contextName, gvr, namespace, name string) error {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return err
	}

	policy := metav1.DeletePropagationBackground
	return client.Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func (e *ResourceEngine) ForceDelete(ctx context.Context, contextName, gvr, namespace, name string) error {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return err
	}

	grace := int64(0)
	policy := metav1.DeletePropagationBackground
	return client.Delete(ctx, name, metav1.DeleteOptions{
		GracePeriodSeconds: &grace,
		PropagationPolicy:  &policy,
	})
}

func (e *ResourceEngine) Create(ctx context.Context, contextName, gvr, namespace string, obj map[string]any) (map[string]any, error) {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{Object: obj}
	created, err := client.Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating %s: %w", gvr, err)
	}

	e.enrich(gvr, created)
	return created.Object, nil
}

func (e *ResourceEngine) Update(ctx context.Context, contextName, gvr, namespace string, obj map[string]any) (map[string]any, error) {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{Object: obj}
	updated, err := client.Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating %s/%s: %w", gvr, u.GetName(), err)
	}

	e.enrich(gvr, updated)
	return updated.Object, nil
}

func (e *ResourceEngine) Patch(ctx context.Context, contextName, gvr, namespace, name string, patchType types.PatchType, data []byte) (map[string]any, error) {
	client, err := e.client(contextName, gvr, namespace)
	if err != nil {
		return nil, err
	}

	updated, err := client.Patch(ctx, name, patchType, data, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("patching %s/%s: %w", gvr, name, err)
	}

	e.enrich(gvr, updated)
	return updated.Object, nil
}
