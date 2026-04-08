package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestWebhookConfigEnricher_MutatingWebhookCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"webhooks": []any{
			map[string]any{"name": "webhook1.example.com"},
			map[string]any{"name": "webhook2.example.com"},
		},
	}}
	e := &enrichers.WebhookConfigEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "webhookCount")
	testza.AssertEqual(t, "2", count)
}

func TestWebhookConfigEnricher_ValidatingWebhookCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"webhooks": []any{
			map[string]any{"name": "validating.example.com"},
		},
	}}
	e := &enrichers.WebhookConfigEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "webhookCount")
	testza.AssertEqual(t, "1", count)
}

func TestWebhookConfigEnricher_NoWebhooks(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	e := &enrichers.WebhookConfigEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "webhookCount")
	testza.AssertEqual(t, "0", count)
}
