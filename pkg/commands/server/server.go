package server

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/generated/controllers/admissionregistration.k8s.io"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/start"

	"github.com/ekristen/cloud-provider-zero/pkg/common"
	"github.com/ekristen/cloud-provider-zero/pkg/webhook"
)

func Execute(c *cli.Context) error {
	ctx, cancel := context.WithCancel(c.Context)
	defer cancel()

	cfg, err := kubeconfig.GetNonInteractiveClientConfig(c.String("kubeconfig")).ClientConfig()
	if err != nil {
		return err
	}

	applyFactory, err := apply.NewForConfig(cfg)
	if err != nil {
		return err
	}

	coreFactory, err := core.NewFactoryFromConfig(cfg)
	if err != nil {
		return err
	}

	admission, err := admissionregistration.NewFactoryFromConfig(cfg)
	if err != nil {
		return err
	}

	options := webhook.NewOptions()
	options.Namespace = c.String("namespace")
	options.DevelopmentBaseURL = c.String("development-base-url")

	if err := webhook.Start(ctx, applyFactory, coreFactory.Core().V1().Secret(), options); err != nil {
		return err
	}

	if err := start.All(ctx, 10, coreFactory, admission); err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

func init() {
	flags := []cli.Flag{
		&cli.IntFlag{
			Name:    "port",
			Usage:   "Port for the HTTP Server Port",
			EnvVars: []string{"PORT"},
			Value:   9443,
		},
		&cli.StringFlag{
			Name:    "development-base-url",
			Aliases: []string{"dev-base-url"},
			EnvVars: []string{"DEVELOPMENT_BASE_URL"},
		},
		&cli.StringFlag{
			Name:  "namespace",
			Value: "cloud-provider-zero",
		},
	}

	cmd := &cli.Command{
		Name:        "webhook-server",
		Usage:       "start the webhook server for validations and mutations",
		Description: `runs a webhook server for cloud provider zero`,
		Before:      common.Before,
		Flags:       append(common.Flags(), flags...),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
