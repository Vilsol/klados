package volumebrowser

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/cluster"
)

// ScanOrphans lists pods cluster-wide with the pvc-browser label and returns
// those whose session label does NOT match the current sessionUUID.
func ScanOrphans(ctx context.Context, conn *cluster.Connection, contextName, sessionUUID string) ([]OrphanPod, error) {
	selector := fmt.Sprintf("%s=%s", LabelPurpose, PurposeValue)
	list, err := conn.Dynamic.Resource(podGVR).Namespace("").List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing pvc-browser pods: %w", err)
	}

	var orphans []OrphanPod
	for _, pod := range list.Items {
		labels := pod.GetLabels()
		session := labels[LabelSession]
		if session == sessionUUID {
			continue
		}
		orphans = append(orphans, OrphanPod{
			ContextName: contextName,
			Namespace:   pod.GetNamespace(),
			PodName:     pod.GetName(),
			PVCName:     labels[LabelPVC],
			CreatedAt:   creationTimestamp(&pod),
			SessionUUID: session,
		})
	}
	return orphans, nil
}

func creationTimestamp(pod *unstructured.Unstructured) time.Time {
	ts := pod.GetCreationTimestamp()
	return ts.Time
}
