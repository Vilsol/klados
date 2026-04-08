package enrichers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type HPAEnricher struct{}

func (e *HPAEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	kind, _, _ := unstructured.NestedString(obj.Object, "spec", "scaleTargetRef", "kind")
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "scaleTargetRef", "name")
	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%s/%s", kind, name), "status", "referenceDisplay")

	specMetrics, _, _ := unstructured.NestedSlice(obj.Object, "spec", "metrics")
	currentMetrics, _, _ := unstructured.NestedSlice(obj.Object, "status", "currentMetrics")

	currentByName := map[string]map[string]any{}
	for _, cm := range currentMetrics {
		cmap, ok := cm.(map[string]any)
		if !ok {
			continue
		}
		metricType, _ := cmap["type"].(string)
		switch metricType {
		case "Resource":
			if res, ok := cmap["resource"].(map[string]any); ok {
				if n, ok := res["name"].(string); ok {
					currentByName["Resource:"+n] = cmap
				}
			}
		case "Pods":
			if pods, ok := cmap["pods"].(map[string]any); ok {
				if metric, ok := pods["metric"].(map[string]any); ok {
					if n, ok := metric["name"].(string); ok {
						currentByName["Pods:"+n] = cmap
					}
				}
			}
		case "Object":
			if object, ok := cmap["object"].(map[string]any); ok {
				if metric, ok := object["metric"].(map[string]any); ok {
					if n, ok := metric["name"].(string); ok {
						currentByName["Object:"+n] = cmap
					}
				}
			}
		case "External":
			if ext, ok := cmap["external"].(map[string]any); ok {
				if metric, ok := ext["metric"].(map[string]any); ok {
					if n, ok := metric["name"].(string); ok {
						currentByName["External:"+n] = cmap
					}
				}
			}
		}
	}

	var parts []string
	for _, m := range specMetrics {
		mmap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		metricType, _ := mmap["type"].(string)
		var part string
		switch metricType {
		case "Resource":
			res, _ := mmap["resource"].(map[string]any)
			metricName, _ := res["name"].(string)
			target, _ := res["target"].(map[string]any)
			targetVal := targetValue(target)
			currentVal := "?"
			if cm, ok := currentByName["Resource:"+metricName]; ok {
				if cmRes, ok := cm["resource"].(map[string]any); ok {
					currentVal = currentResourceValue(cmRes)
				}
			}
			part = fmt.Sprintf("%s: %s/%s", metricName, currentVal, targetVal)
		case "Pods":
			pods, _ := mmap["pods"].(map[string]any)
			metric, _ := pods["metric"].(map[string]any)
			metricName, _ := metric["name"].(string)
			target, _ := pods["target"].(map[string]any)
			targetVal := targetValue(target)
			currentVal := "?"
			if cm, ok := currentByName["Pods:"+metricName]; ok {
				if cmPods, ok := cm["pods"].(map[string]any); ok {
					if cur, ok := cmPods["current"].(map[string]any); ok {
						currentVal = anyString(cur["averageValue"])
					}
				}
			}
			part = fmt.Sprintf("%s: %s/%s", metricName, currentVal, targetVal)
		case "Object":
			object, _ := mmap["object"].(map[string]any)
			metric, _ := object["metric"].(map[string]any)
			metricName, _ := metric["name"].(string)
			target, _ := object["target"].(map[string]any)
			targetVal := targetValue(target)
			currentVal := "?"
			if cm, ok := currentByName["Object:"+metricName]; ok {
				if cmObj, ok := cm["object"].(map[string]any); ok {
					if cur, ok := cmObj["current"].(map[string]any); ok {
						currentVal = anyString(cur["value"])
					}
				}
			}
			part = fmt.Sprintf("%s: %s/%s", metricName, currentVal, targetVal)
		case "External":
			ext, _ := mmap["external"].(map[string]any)
			metric, _ := ext["metric"].(map[string]any)
			metricName, _ := metric["name"].(string)
			target, _ := ext["target"].(map[string]any)
			targetVal := targetValue(target)
			currentVal := "?"
			if cm, ok := currentByName["External:"+metricName]; ok {
				if cmExt, ok := cm["external"].(map[string]any); ok {
					if cur, ok := cmExt["current"].(map[string]any); ok {
						currentVal = anyString(cur["averageValue"])
					}
				}
			}
			part = fmt.Sprintf("%s: %s/%s", metricName, currentVal, targetVal)
		default:
			part = metricType
		}
		parts = append(parts, part)
		if len(parts) == 3 {
			break
		}
	}

	display := strings.Join(parts, ", ")
	if len(specMetrics) > 3 {
		display += ", ..."
	}
	_ = unstructured.SetNestedField(obj.Object, display, "status", "targetsDisplay")

	return nil
}

func targetValue(target map[string]any) string {
	if target == nil {
		return "?"
	}
	if v, ok := target["averageUtilization"]; ok {
		return fmt.Sprintf("%v%%", v)
	}
	if v, ok := target["averageValue"]; ok {
		return anyString(v)
	}
	if v, ok := target["value"]; ok {
		return anyString(v)
	}
	return "?"
}

func currentResourceValue(res map[string]any) string {
	if v, ok := res["currentAverageUtilization"]; ok {
		return fmt.Sprintf("%v%%", v)
	}
	if v, ok := res["currentAverageValue"]; ok {
		return anyString(v)
	}
	return "?"
}

func anyString(v any) string {
	if v == nil {
		return "?"
	}
	return fmt.Sprintf("%v", v)
}
