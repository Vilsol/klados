//go:build !release

package services

import (
	"context"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/klados/internal/resource"
	"k8s.io/client-go/kubernetes"
)

// NewResourceServiceForTest creates a ResourceService for unit testing
// by injecting a fake Kubernetes clientset.
func NewResourceServiceForTest(
	clientset kubernetes.Interface,
	engine *resource.ResourceEngine,
	reg *resource.Registry,
	enricherReg *resource.EnricherRegistry,
) *ResourceService {
	mgr := cluster.NewManager(func(string, any) {}, &config.Config{}, context.Background())
	conn := &cluster.Connection{Clientset: clientset}
	mgr.SetConnectionForTest("ctx", conn)

	appSvc := &AppService{clusterMgr: mgr}
	return &ResourceService{
		appService:  appSvc,
		engine:      engine,
		registry:    reg,
		enricherReg: enricherReg,
		ctx:         context.Background(),
	}
}
