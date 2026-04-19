package volumebrowser

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/cluster"
)

// ScanOrphans lists pods cluster-wide that belong to the same klados user on
// the same host (matched via labels), and returns the subset whose session
// label does NOT match the current sessionUUID. Scoping by host+user ensures
// that two klados instances pointed at the same cluster by different humans
// never see each other's live pods as orphans.
func ScanOrphans(ctx context.Context, conn *cluster.Connection, contextName, sessionUUID, hostLabel, userLabel string) ([]OrphanPod, error) {
	selector := fmt.Sprintf("%s=%s,%s=%s,%s=%s",
		LabelPurpose, PurposeValue,
		LabelHost, hostLabel,
		LabelUser, userLabel,
	)
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
