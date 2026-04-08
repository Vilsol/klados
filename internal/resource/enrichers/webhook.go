package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type WebhookConfigEnricher struct{}

func (e *WebhookConfigEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	webhooks, _, _ := unstructured.NestedSlice(obj.Object, "webhooks")
	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d", len(webhooks)), "status", "webhookCount")
	return nil
}
