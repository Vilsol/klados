package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	dynfake "k8s.io/client-go/dynamic/fake"
	kfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/resource"
	"github.com/Vilsol/klados/internal/services"
)

func newTestResourceService(objects ...runtime.Object) (*services.ResourceService, *kfake.Clientset) {
	cs := kfake.NewSimpleClientset(objects...)
	dynCS := dynfake.NewSimpleDynamicClient(scheme.Scheme, objects...)
	reg, _ := resource.NewRegistry()
	enricherReg := resource.NewEnricherRegistry()
	_ = resource.RegisterBuiltin(reg, enricherReg, nil)
	conn := &cluster.Connection{Clientset: cs, Dynamic: dynCS}
	provider := &testConnProvider{conn: conn}
	engine := resource.NewResourceEngine(provider, enricherReg)
	svc := services.NewResourceServiceForTest(cs, engine, reg, enricherReg)
	return svc, cs
}

type testConnProvider struct {
	conn *cluster.Connection
}

func (p *testConnProvider) GetConnection(_ string) (*cluster.Connection, error) {
	return p.conn, nil
}

func TestResourceService_DeleteJobCascade(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "myjob", Namespace: "default"},
	}
	svc, cs := newTestResourceService(job)
	err := svc.DeleteJobCascade("ctx", "default", "myjob")
	testza.AssertNoError(t, err)
	_, err = cs.BatchV1().Jobs("default").Get(context.Background(), "myjob", metav1.GetOptions{})
	testza.AssertNotNil(t, err) // should be gone
}

func TestResourceService_DeleteJobOrphan(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "myjob", Namespace: "default"},
	}
	svc, cs := newTestResourceService(job)
	err := svc.DeleteJobOrphan("ctx", "default", "myjob")
	testza.AssertNoError(t, err)
	_, err = cs.BatchV1().Jobs("default").Get(context.Background(), "myjob", metav1.GetOptions{})
	testza.AssertNotNil(t, err)
}

func TestResourceService_TriggerCronJob(t *testing.T) {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "mycron", Namespace: "default"},
		Spec: batchv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "c", Image: "img"}},
						},
					},
				},
			},
		},
	}
	svc, cs := newTestResourceService(cj)
	err := svc.TriggerCronJob("ctx", "default", "mycron")
	testza.AssertNoError(t, err)

	jobs, err := cs.BatchV1().Jobs("default").List(context.Background(), metav1.ListOptions{})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(jobs.Items))
	testza.AssertTrue(t, len(jobs.Items[0].Name) <= 63)
}

func TestResourceService_TriggerCronJob_LongName(t *testing.T) {
	longName := "this-is-a-very-long-cronjob-name-that-exceeds-48-characters-total"
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: longName, Namespace: "default"},
		Spec: batchv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "c", Image: "img"}},
						},
					},
				},
			},
		},
	}
	svc, cs := newTestResourceService(cj)
	err := svc.TriggerCronJob("ctx", "default", longName)
	testza.AssertNoError(t, err)

	jobs, err := cs.BatchV1().Jobs("default").List(context.Background(), metav1.ListOptions{})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(jobs.Items))
	testza.AssertTrue(t, len(jobs.Items[0].Name) <= 63)
}

func TestResourceService_SuspendCronJob(t *testing.T) {
	suspend := false
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "mycron", Namespace: "default"},
		Spec:       batchv1.CronJobSpec{Suspend: &suspend},
	}
	svc, _ := newTestResourceService(cj)
	// Dynamic fake client accepts patch; correctness verified by integration tests.
	err := svc.SuspendCronJob("ctx", "default", "mycron")
	testza.AssertNoError(t, err)
}

func TestResourceService_ResumeCronJob(t *testing.T) {
	suspend := true
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "mycron", Namespace: "default"},
		Spec:       batchv1.CronJobSpec{Suspend: &suspend},
	}
	svc, _ := newTestResourceService(cj)
	err := svc.ResumeCronJob("ctx", "default", "mycron")
	testza.AssertNoError(t, err)
}

func TestResourceService_GetRolloutHistory_Deployment(t *testing.T) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "default", UID: "uid-abc"},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myapp-rs1",
			Namespace: "default",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "3",
			},
			OwnerReferences: []metav1.OwnerReference{
				{UID: types.UID("uid-abc")},
			},
		},
	}
	svc, _ := newTestResourceService(deploy, rs)
	history, err := svc.GetRolloutHistory("ctx", "apps.v1.deployments", "default", "myapp")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(history))
	testza.AssertEqual(t, int64(3), history[0].Revision)
}

func TestResourceService_PauseRollout(t *testing.T) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{},
	}
	svc, _ := newTestResourceService(deploy)
	err := svc.PauseRollout("ctx", "default", "myapp")
	testza.AssertNoError(t, err)
}

func TestResourceService_ResumeRollout(t *testing.T) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Paused: true},
	}
	svc, _ := newTestResourceService(deploy)
	err := svc.ResumeRollout("ctx", "default", "myapp")
	testza.AssertNoError(t, err)
}

func TestRolloutHistory_IgnoresUnownedRS(t *testing.T) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "default", UID: "uid-abc"},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-rs",
			Namespace: "default",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{UID: types.UID("uid-other")},
			},
		},
	}
	svc, _ := newTestResourceService(deploy, rs)
	history, err := svc.GetRolloutHistory("ctx", "apps.v1.deployments", "default", "myapp")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 0, len(history))
}

// jsonRaw is a helper to build a runtime.RawExtension.
func jsonRaw(v any) runtime.RawExtension {
	data, _ := json.Marshal(v)
	return runtime.RawExtension{Raw: data}
}

func TestResourceService_GetRolloutHistory_StatefulSet(t *testing.T) {
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "myss", Namespace: "default", UID: "uid-ss"},
	}
	cr := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myss-cr1",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{UID: types.UID("uid-ss")},
			},
		},
		Revision: 2,
		Data:     jsonRaw(map[string]any{}),
	}
	svc, _ := newTestResourceService(ss, cr)
	history, err := svc.GetRolloutHistory("ctx", "apps.v1.statefulsets", "default", "myss")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(history))
	testza.AssertEqual(t, int64(2), history[0].Revision)
}

// ensure time reference doesn't cause unused import
var _ = time.Now
