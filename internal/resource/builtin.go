package resource

import (
	"fmt"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

var builtinDescriptors = []*Descriptor{
	{
		Group: "", Version: "v1", Resource: "pods", Kind: "Pod",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Status", Expr: "status.statusDisplay", RenderType: RenderBadge, Width: 100},
			{Name: "Restarts", Expr: "status.restartCount", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Node", Expr: "spec.nodeName", RenderType: RenderText},
			{Label: "Status", Expr: "status.statusDisplay", RenderType: RenderBadge},
			{Label: "Pod IP", Expr: "status.podIP", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Restarts", Expr: "status.restartCount", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "containers", "logs", "terminal", "labels", "events", "yaml"},
		Actions:      []string{"delete", "force-delete"},
	},
	{
		Group: "apps", Version: "v1", Resource: "deployments", Kind: "Deployment",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Available", Expr: "status.availableReplicas", RenderType: RenderText, Width: 90},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Replicas", Expr: "status.replicas", RenderType: RenderText},
			{Label: "Available", Expr: "status.availableReplicas", RenderType: RenderText},
			{Label: "Strategy", Expr: "spec.strategy.type", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"scale", "restart", "delete"},
	},
	{
		Group: "apps", Version: "v1", Resource: "statefulsets", Kind: "StatefulSet",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Replicas", Expr: "status.replicas", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"scale", "restart", "delete"},
	},
	{
		Group: "apps", Version: "v1", Resource: "daemonsets", Kind: "DaemonSet",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Desired", Expr: "status.desiredNumberScheduled", RenderType: RenderText},
			{Label: "Available", Expr: "status.numberAvailable", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"restart", "delete"},
	},
	{
		Group: "apps", Version: "v1", Resource: "replicasets", Kind: "ReplicaSet",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Ready", Expr: "status.readyReplicas", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyReplicas", RenderType: RenderText},
			{Label: "Replicas", Expr: "status.replicas", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "batch", Version: "v1", Resource: "jobs", Kind: "Job",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Completions", Expr: "status.completionDisplay", RenderType: RenderText, Width: 100},
			{Name: "Duration", Expr: "status.durationDisplay", RenderType: RenderText, Width: 90},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Completions", Expr: "status.completionDisplay", RenderType: RenderText},
			{Label: "Duration", Expr: "status.durationDisplay", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "batch", Version: "v1", Resource: "cronjobs", Kind: "CronJob",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Schedule", Expr: "spec.schedule", RenderType: RenderText, Width: 120},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Schedule", Expr: "spec.schedule", RenderType: RenderText},
			{Label: "Suspend", Expr: "spec.suspend", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "", Version: "v1", Resource: "services", Kind: "Service",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Type", Expr: "spec.type", RenderType: RenderBadge, Width: 100},
			{Name: "Cluster IP", Expr: "spec.clusterIP", RenderType: RenderText, Width: 130},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Type", Expr: "spec.type", RenderType: RenderBadge},
			{Label: "Cluster IP", Expr: "spec.clusterIP", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "service", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "networking.k8s.io", Version: "v1", Resource: "ingresses", Kind: "Ingress",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "ingress", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "", Version: "v1", Resource: "configmaps", Kind: "ConfigMap",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "configmap", "labels", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "", Version: "v1", Resource: "secrets", Kind: "Secret",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Type", Expr: "type", RenderType: RenderBadge, Width: 160},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "secret", "labels", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "", Version: "v1", Resource: "persistentvolumes", Kind: "PersistentVolume",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Status", Expr: "status.phase", RenderType: RenderBadge, Width: 100},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Status", Expr: "status.phase", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
	{
		Group: "", Version: "v1", Resource: "nodes", Kind: "Node",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Status", Expr: "status.readyStatus", RenderType: RenderBadge, Width: 90},
			{Name: "Roles", Expr: "status.roles", RenderType: RenderText, Width: 130},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Status", Expr: "status.readyStatus", RenderType: RenderBadge},
			{Label: "Roles", Expr: "status.roles", RenderType: RenderText},
			{Label: "Conditions", Expr: "status.conditionsSummary", RenderType: RenderText},
			{Label: "Taints", Expr: "status.taintsSummary", RenderType: RenderText},
			{Label: "OS Image", Expr: "status.nodeInfo.osImage", RenderType: RenderText},
			{Label: "Kernel", Expr: "status.nodeInfo.kernelVersion", RenderType: RenderText},
			{Label: "Container Runtime", Expr: "status.nodeInfo.containerRuntimeVersion", RenderType: RenderText},
			{Label: "CPU Allocatable", Expr: "status.allocatable.cpu", RenderType: RenderText},
			{Label: "Memory Allocatable", Expr: "status.allocatable.memory", RenderType: RenderText},
			{Label: "Pods Allocatable", Expr: "status.allocatable.pods", RenderType: RenderText},
			{Label: "Ephemeral Storage", Expr: "status.ephemeralStorage", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "node", "labels", "events", "yaml"},
		Actions:      []string{},
	},
	{
		Group: "", Version: "v1", Resource: "persistentvolumeclaims", Kind: "PersistentVolumeClaim",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Status", Expr: "status.phase", RenderType: RenderBadge, Width: 100},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Status", Expr: "status.phase", RenderType: RenderBadge},
			{Label: "Volume", Expr: "spec.volumeName", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []string{"delete"},
	},
}

func RegisterBuiltin(reg *Registry, enricherReg *EnricherRegistry) error {
	for _, d := range builtinDescriptors {
		if err := reg.Register(d); err != nil {
			return fmt.Errorf("registering %s: %w", d.GVR(), err)
		}
	}

	enricherReg.Register("core.v1.pods", &enrichers.PodEnricher{})
	enricherReg.Register("apps.v1.deployments", &enrichers.DeploymentEnricher{})
	enricherReg.Register("apps.v1.statefulsets", &enrichers.StatefulSetEnricher{})
	enricherReg.Register("apps.v1.daemonsets", &enrichers.DaemonSetEnricher{})
	enricherReg.Register("batch.v1.jobs", &enrichers.JobEnricher{})
	enricherReg.Register("core.v1.nodes", &enrichers.NodeEnricher{})

	return nil
}
