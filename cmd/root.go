package cmd

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/klados/internal/logging"
	"github.com/Vilsol/klados/internal/services"
	"github.com/Vilsol/klados/internal/session"
	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func Execute(assets embed.FS) {
	root := &cobra.Command{
		Use:           "klados",
		Short:         "Kubernetes Desktop IDE",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := logging.Setup()

			cfg, err := config.Load()
			if err != nil {
				log.Fatal("failed to load config:", err)
			}

			sess, err := session.Load()
			if err != nil {
				log.Fatal("failed to load session:", err)
			}

			appSvc := services.NewAppService(cfg, sess, ctx)
			clusterSvc := services.NewClusterService(appSvc, sess)
			configSvc := services.NewConfigService(ctx, cfg)
			drainSvc := services.NewDrainService(appSvc)
			resourceSvc := services.NewResourceService(appSvc, drainSvc)
			schemaSvc := services.NewSchemaService(appSvc)
			logSvc := services.NewLogService(appSvc)
			execSvc := services.NewExecService(appSvc)
			portForwardSvc := services.NewPortForwardService(appSvc)
			volumeBrowserSvc := services.NewVolumeBrowserService(appSvc)
			clusterSvc.SetVolumeBrowserService(volumeBrowserSvc)
			appSvc.SetVolumeBrowserService(volumeBrowserSvc)
			metricsSvc := services.NewMetricsService(appSvc)
			pluginSvc := services.NewPluginService(appSvc, resourceSvc)
			windowSvc := services.NewWindowService()
			appSvc.SetPluginService(pluginSvc)
			metricsSvc.SetPluginService(pluginSvc)

			app := application.New(application.Options{
				Name:        "klados",
				Description: "Kubernetes Desktop IDE",
				Services: []application.Service{
					application.NewService(appSvc),
					application.NewService(clusterSvc),
					application.NewService(configSvc),
					application.NewService(resourceSvc),
					application.NewService(schemaSvc),
					application.NewService(logSvc),
					application.NewService(execSvc),
					application.NewService(portForwardSvc),
					application.NewService(volumeBrowserSvc),
					application.NewService(drainSvc),
					application.NewService(metricsSvc),
					application.NewService(pluginSvc),
					application.NewService(windowSvc),
				},
				Assets: application.AssetOptions{
					Handler: application.AssetFileServerFS(assets),
				},
				Mac: application.MacOptions{
					ApplicationShouldTerminateAfterLastWindowClosed: true,
				},
			})

			winOpts := application.WebviewWindowOptions{
				Title:           "Klados",
				DevToolsEnabled: true,
				Mac: application.MacWindow{
					InvisibleTitleBarHeight: 50,
					Backdrop:               application.MacBackdropTranslucent,
					TitleBar:               application.MacTitleBarHiddenInset,
				},
				BackgroundColour: application.NewRGB(27, 38, 54),
				URL:              "/",
			}

			if sess.Window.Width > 0 {
				winOpts.Width = sess.Window.Width
			}
			if sess.Window.Height > 0 {
				winOpts.Height = sess.Window.Height
			}
			if sess.Window.X > 0 || sess.Window.Y > 0 {
				winOpts.X = sess.Window.X
				winOpts.Y = sess.Window.Y
			}

			app.Window.NewWithOptions(winOpts)

			return app.Run()
		},
	}

	root.AddCommand(pluginCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
