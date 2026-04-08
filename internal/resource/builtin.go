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
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Status", Expr: "status.statusDisplay", RenderType: RenderBadge, Width: 100},
			{Name: "Restarts", Expr: "status.restartCount", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Node", Expr: "spec.nodeName", RenderType: RenderText, Hidden: true},
			{Name: "IP", Expr: "status.podIP", RenderType: RenderText, Hidden: true},
			{Name: "QoS", Expr: "status.qosClass", RenderType: RenderBadge, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Node", Expr: "spec.nodeName", RenderType: RenderText},
			{Label: "Status", Expr: "status.statusDisplay", RenderType: RenderBadge},
			{Label: "Pod IP", Expr: "status.podIP", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "containers", "logs", "terminal", "labels", "events", "metrics", "yaml"},
		Actions: []Action{
			{Name: "delete", Label: "Delete"},
			{Name: "force-delete", Label: "Force Delete"},
		},
	},
	{
		Group: "apps", Version: "v1", Resource: "deployments", Kind: "Deployment",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Available", Expr: "status.availableReplicas", RenderType: RenderText, Width: 90},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Up-to-date", Expr: "status.updatedReplicas", RenderType: RenderText, Hidden: true},
			{Name: "Strategy", Expr: "spec.strategy.type", RenderType: RenderBadge, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Replicas", Expr: "status.replicas", RenderType: RenderText},
			{Label: "Available", Expr: "status.availableReplicas", RenderType: RenderText},
			{Label: "Strategy", Expr: "spec.strategy.type", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "metrics", "yaml"},
		Actions: []Action{
			{Name: "pause", Label: "Pause Rollout", DisabledWhen: "spec.paused == true", DisabledReason: "Rollout is already paused"},
			{Name: "resume", Label: "Resume Rollout", DisabledWhen: "spec.paused != true", DisabledReason: "Rollout is not paused"},
			{Name: "rollback", Label: "Rollback"},
			{Name: "scale", Label: "Scale"},
			{Name: "restart", Label: "Restart"},
			{Name: "delete", Label: "Delete"},
		},
	},
	{
		Group: "apps", Version: "v1", Resource: "statefulsets", Kind: "StatefulSet",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Available", Expr: "status.availableReplicas", RenderType: RenderText, Hidden: true},
			{Name: "Current", Expr: "status.currentReplicas", RenderType: RenderText, Hidden: true},
			{Name: "Updated", Expr: "status.updatedReplicas", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Replicas", Expr: "status.replicas", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "metrics", "yaml"},
		Actions: []Action{
			{Name: "rollback", Label: "Rollback"},
			{Name: "scale", Label: "Scale"},
			{Name: "restart", Label: "Restart"},
			{Name: "delete", Label: "Delete"},
		},
	},
	{
		Group: "apps", Version: "v1", Resource: "daemonsets", Kind: "DaemonSet",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Ready", Expr: "status.readyDisplay", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Desired", Expr: "status.desiredNumberScheduled", RenderType: RenderText, Hidden: true},
			{Name: "Available", Expr: "status.numberAvailable", RenderType: RenderText, Hidden: true},
			{Name: "Node Selector", Expr: "status.nodeSelectorDisplay", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyDisplay", RenderType: RenderText},
			{Label: "Desired", Expr: "status.desiredNumberScheduled", RenderType: RenderText},
			{Label: "Available", Expr: "status.numberAvailable", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "metrics", "yaml"},
		Actions: []Action{
			{Name: "rollback", Label: "Rollback"},
			{Name: "restart", Label: "Restart"},
			{Name: "delete", Label: "Delete"},
		},
	},
	{
		Group: "apps", Version: "v1", Resource: "replicasets", Kind: "ReplicaSet",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Ready", Expr: "status.readyReplicas", RenderType: RenderText, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Replicas", Expr: "status.replicas", RenderType: RenderText, Hidden: true},
			{Name: "Owner", Expr: "status.ownerDisplay", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Ready", Expr: "status.readyReplicas", RenderType: RenderText},
			{Label: "Replicas", Expr: "status.replicas", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "batch", Version: "v1", Resource: "jobs", Kind: "Job",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Completions", Expr: "status.completionDisplay", RenderType: RenderText, Width: 100},
			{Name: "Duration", Expr: "status.durationDisplay", RenderType: RenderText, Width: 90},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Status", Expr: "status.statusDisplay", RenderType: RenderBadge, Hidden: true},
			{Name: "Backoff Limit", Expr: "spec.backoffLimit", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Completions", Expr: "status.completionDisplay", RenderType: RenderText},
			{Label: "Duration", Expr: "status.durationDisplay", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions: []Action{
			{Name: "delete-cascade", Label: "Delete (Cascade)"},
			{Name: "delete-orphan", Label: "Delete (Orphan Pods)"},
			{Name: "delete", Label: "Delete"},
		},
	},
	{
		Group: "batch", Version: "v1", Resource: "cronjobs", Kind: "CronJob",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Schedule", Expr: "spec.schedule", RenderType: RenderText, Width: 120},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Suspend", Expr: "spec.suspend", RenderType: RenderBadge, Hidden: true},
			{Name: "Last Schedule", Expr: "status.lastScheduleTime", RenderType: RenderAge, Hidden: true},
			{Name: "Active", Expr: "status.activeCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Schedule", Expr: "spec.schedule", RenderType: RenderText},
			{Label: "Suspend", Expr: "spec.suspend", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions: []Action{
			{Name: "trigger", Label: "Trigger Now"},
			{Name: "suspend", Label: "Suspend", DisabledWhen: "spec.suspend == true", DisabledReason: "CronJob is already suspended"},
			{Name: "resume", Label: "Resume", DisabledWhen: "spec.suspend != true", DisabledReason: "CronJob is not suspended"},
			{Name: "delete", Label: "Delete"},
		},
	},
	{
		Group: "", Version: "v1", Resource: "services", Kind: "Service",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Type", Expr: "spec.type", RenderType: RenderBadge, Width: 100},
			{Name: "Cluster IP", Expr: "spec.clusterIP", RenderType: RenderText, Width: 130},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "External IP", Expr: "status.externalIPDisplay", RenderType: RenderText, Hidden: true},
			{Name: "Ports", Expr: "status.portsDisplay", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Type", Expr: "spec.type", RenderType: RenderBadge},
			{Label: "Cluster IP", Expr: "spec.clusterIP", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "service", "labels", "events", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "networking.k8s.io", Version: "v1", Resource: "ingresses", Kind: "Ingress",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Class", Expr: "spec.ingressClassName", RenderType: RenderText, Hidden: true},
			{Name: "Hosts", Expr: "status.hostsDisplay", RenderType: RenderText, Hidden: true},
			{Name: "Default Backend", Expr: "status.defaultBackendDisplay", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "ingress", "labels", "events", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "configmaps", Kind: "ConfigMap",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Keys", Expr: "status.dataKeysCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "configmap", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "secrets", Kind: "Secret",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Type", Expr: "type", RenderType: RenderBadge, Width: 160},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Keys", Expr: "status.dataKeysCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "secret", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "persistentvolumes", Kind: "PersistentVolume",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Status", Expr: "status.phase", RenderType: RenderBadge, Width: 100},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Capacity", Expr: "spec.capacity.storage", RenderType: RenderText, Hidden: true},
			{Name: "Access Modes", Expr: "status.accessModesDisplay", RenderType: RenderText, Hidden: true},
			{Name: "Storage Class", Expr: "spec.storageClassName", RenderType: RenderText, Hidden: true},
			{Name: "Claim", Expr: "status.claimDisplay", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Status", Expr: "status.phase", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "namespaces", Kind: "Namespace",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Status", Expr: "status.phase", RenderType: RenderBadge, Width: 90},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Status", Expr: "status.phase", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "nodes", Kind: "Node",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Status", Expr: "status.readyStatus", RenderType: RenderBadge, Width: 90},
			{Name: "Drain", Expr: "status.drainPhase", RenderType: RenderBadge, Width: 90},
			{Name: "Roles", Expr: "status.roles", RenderType: RenderText, Width: 130},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Version", Expr: "status.nodeInfo.kubeletVersion", RenderType: RenderText, Hidden: true},
			{Name: "Internal IP", Expr: "status.internalIPDisplay", RenderType: RenderText, Hidden: true},
			{Name: "OS/Arch", Expr: "status.osArchDisplay", RenderType: RenderText, Hidden: true},
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
		DetailPanels: []string{"overview", "node", "drain", "labels", "events", "metrics", "yaml"},
		Actions: []Action{
			{Name: "cordon", Label: "Cordon", DisabledWhen: "spec.unschedulable == true", DisabledReason: "Node is already cordoned"},
			{Name: "uncordon", Label: "Uncordon", DisabledWhen: "spec.unschedulable != true", DisabledReason: "Node is not cordoned"},
			{Name: "drain", Label: "Drain", DisabledWhen: "status.drainPhase == 'Draining'", DisabledReason: "Node is already draining"},
			{Name: "delete", Label: "Delete"},
		},
	},
	{
		Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions", Kind: "CustomResourceDefinition",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Group", Expr: "spec.group", RenderType: RenderText},
			{Name: "Scope", Expr: "spec.scope", RenderType: RenderBadge, Width: 110},
			{Name: "Versions", Expr: "status.versionsDisplay", RenderType: RenderText},
			{Name: "Established", Expr: "status.established", RenderType: RenderBadge, Width: 110},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Group", Expr: "spec.group", RenderType: RenderText},
			{Label: "Scope", Expr: "spec.scope", RenderType: RenderBadge},
			{Label: "Plural", Expr: "spec.names.plural", RenderType: RenderText},
			{Label: "Singular", Expr: "spec.names.singular", RenderType: RenderText},
			{Label: "Kind", Expr: "spec.names.kind", RenderType: RenderText},
			{Label: "Short Names", Expr: "spec.names.shortNames.join(', ')", RenderType: RenderText},
			{Label: "Storage Version", Expr: "status.storageVersion", RenderType: RenderText},
			{Label: "Established", Expr: "status.established", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "crd", "crd-schema", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "serviceaccounts", Kind: "ServiceAccount",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Secrets", Expr: "status.secretsCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "serviceaccount", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles", Kind: "Role",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Rules", Expr: "status.rulesCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "rules", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles", Kind: "ClusterRole",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Rules", Expr: "status.rulesCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "rules", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings", Kind: "RoleBinding",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Role", Expr: "status.roleRefDisplay", RenderType: RenderText, Hidden: true},
			{Name: "Subjects", Expr: "status.subjectsCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "binding", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings", Kind: "ClusterRoleBinding",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Role", Expr: "status.roleRefDisplay", RenderType: RenderText, Hidden: true},
			{Name: "Subjects", Expr: "status.subjectsCount", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "binding", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "", Version: "v1", Resource: "persistentvolumeclaims", Kind: "PersistentVolumeClaim",
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150},
			{Name: "Status", Expr: "status.phase", RenderType: RenderBadge, Width: 100},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Capacity", Expr: "status.capacity.storage", RenderType: RenderText, Hidden: true},
			{Name: "Access Modes", Expr: "status.accessModesDisplay", RenderType: RenderText, Hidden: true},
			{Name: "Storage Class", Expr: "spec.storageClassName", RenderType: RenderText, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Namespace", Expr: "metadata.namespace", RenderType: RenderText},
			{Label: "Status", Expr: "status.phase", RenderType: RenderBadge},
			{Label: "Volume", Expr: "spec.volumeName", RenderType: RenderText},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "labels", "events", "yaml"},
		Actions:      []Action{{Name: "expand", Label: "Expand"}, {Name: "delete", Label: "Delete"}},
	},
	{
		Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses", Kind: "StorageClass",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Provisioner", Expr: "provisioner", RenderType: RenderText},
			{Name: "Reclaim Policy", Expr: "reclaimPolicy", RenderType: RenderBadge, Width: 120},
			{Name: "Binding Mode", Expr: "volumeBindingMode", RenderType: RenderBadge, Width: 140},
			{Name: "Default", Expr: "status.isDefault", RenderType: RenderBadge, Width: 80},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
			{Name: "Allow Expansion", Expr: "allowVolumeExpansion", RenderType: RenderBadge, Hidden: true},
		},
		OverviewFields: []OverviewField{
			{Label: "Provisioner", Expr: "provisioner", RenderType: RenderText},
			{Label: "Reclaim Policy", Expr: "reclaimPolicy", RenderType: RenderBadge},
			{Label: "Binding Mode", Expr: "volumeBindingMode", RenderType: RenderBadge},
			{Label: "Default", Expr: "status.isDefault", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "sc-parameters", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
	{
		Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers", Kind: "CSIDriver",
		ClusterScoped: true,
		Columns: []Column{
			{Name: "Name", Expr: "metadata.name", RenderType: RenderText},
			{Name: "Attach Required", Expr: "string(spec.attachRequired)", RenderType: RenderBadge, Width: 130},
			{Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 80},
		},
		OverviewFields: []OverviewField{
			{Label: "Attach Required", Expr: "string(spec.attachRequired)", RenderType: RenderBadge},
			{Label: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge},
		},
		DetailPanels: []string{"overview", "csi-capabilities", "labels", "yaml"},
		Actions:      []Action{{Name: "delete", Label: "Delete"}},
	},
}

func BuiltinDescriptors() []*Descriptor {
	return builtinDescriptors
}

func RegisterBuiltin(reg *Registry, enricherReg *EnricherRegistry, drainSvc enrichers.DrainStateProvider) error {
	for _, d := range builtinDescriptors {
		if err := reg.Register(d); err != nil {
			return fmt.Errorf("registering %s: %w", d.GVR(), err)
		}
	}

	enricherReg.Register("core.v1.pods", &enrichers.PodEnricher{})
	enricherReg.Register("apps.v1.deployments", &enrichers.DeploymentEnricher{})
	enricherReg.Register("apps.v1.statefulsets", &enrichers.StatefulSetEnricher{})
	enricherReg.Register("apps.v1.daemonsets", &enrichers.DaemonSetEnricher{})
	enricherReg.Register("apps.v1.replicasets", &enrichers.ReplicaSetEnricher{})
	enricherReg.Register("batch.v1.jobs", &enrichers.JobEnricher{})
	enricherReg.Register("batch.v1.cronjobs", &enrichers.CronJobEnricher{})
	enricherReg.Register("core.v1.services", &enrichers.ServiceEnricher{})
	enricherReg.Register("networking.k8s.io.v1.ingresses", &enrichers.IngressEnricher{})
	enricherReg.Register("core.v1.configmaps", &enrichers.ConfigMapEnricher{})
	enricherReg.Register("core.v1.secrets", &enrichers.SecretEnricher{})
	enricherReg.Register("core.v1.persistentvolumes", &enrichers.PVEnricher{})
	enricherReg.Register("core.v1.persistentvolumeclaims", &enrichers.PVCEnricher{})
	enricherReg.Register("core.v1.nodes", &enrichers.NodeEnricher{DrainService: drainSvc})
	enricherReg.Register("core.v1.serviceaccounts", &enrichers.ServiceAccountEnricher{})
	enricherReg.Register("rbac.authorization.k8s.io.v1.roles", &enrichers.RoleEnricher{})
	enricherReg.Register("rbac.authorization.k8s.io.v1.clusterroles", &enrichers.RoleEnricher{})
	enricherReg.Register("rbac.authorization.k8s.io.v1.rolebindings", &enrichers.BindingEnricher{})
	enricherReg.Register("rbac.authorization.k8s.io.v1.clusterrolebindings", &enrichers.BindingEnricher{})
	enricherReg.Register("storage.k8s.io.v1.storageclasses", &enrichers.StorageClassEnricher{})
	enricherReg.Register("apiextensions.k8s.io.v1.customresourcedefinitions", &enrichers.CRDEnricher{})

	return nil
}
