/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/losant/k8s-instrumental-adaptor/pkg/cmd/server"
	"github.com/losant/k8s-instrumental-adaptor/pkg/instrumental_adaptor/provider"
	instrumental "github.com/losant/k8s-instrumental-adaptor/pkg/instrumental_client"
	"github.com/spf13/cobra"
)

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartSampleAdapterServer(out, errOut io.Writer, stopCh <-chan struct{}) *cobra.Command {
	baseOpts := server.NewCustomMetricsAdapterServerOptions(out, errOut)
	o := SampleAdapterServerOptions{
		CustomMetricsAdapterServerOptions: baseOpts,
		DiscoveryInterval:                 10 * time.Minute,
		EnableCustomMetricsAPI:            true,
		EnableExternalMetricsAPI:          true,
	}

	cmd := &cobra.Command{
		Short: "Launch the custom metrics API adapter server",
		Long:  "Launch the custom metrics API adapter server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunCustomMetricsAdapterServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.SecureServing.AddFlags(flags)
	o.Authentication.AddFlags(flags)
	o.Authorization.AddFlags(flags)
	o.Features.AddFlags(flags)

	flags.StringVar(&o.RemoteKubeConfigFile, "lister-kubeconfig", o.RemoteKubeConfigFile, ""+
		"kubeconfig file pointing at the 'core' kubernetes server with enough rights to list "+
		"any described objects")
	flags.DurationVar(&o.DiscoveryInterval, "discovery-interval", o.DiscoveryInterval, ""+
		"interval at which to refresh API discovery information")
	flags.BoolVar(&o.EnableCustomMetricsAPI, "enable-custom-metrics-api", o.EnableCustomMetricsAPI, ""+
		"whether to enable Custom Metrics API")
	flags.BoolVar(&o.EnableExternalMetricsAPI, "enable-external-metrics-api", o.EnableExternalMetricsAPI, ""+
		"whether to enable External Metrics API")

	return cmd
}

func (o SampleAdapterServerOptions) RunCustomMetricsAdapterServer(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	token := os.Getenv("INSTRUMENTAL_TOKEN")
	if token == "" {
		log.Fatal("The provider will not work with an INSTRUMENTAL_TOKEN.")
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	instrumentalClient := instrumental.NewClient(client, token)
	metricsProvider := provider.NewInstrumentalProvider(token, instrumentalClient)
	customMetricsProvider := metricsProvider
	externalMetricsProvider := metricsProvider
	if !o.EnableCustomMetricsAPI {
		customMetricsProvider = nil
	}
	if !o.EnableExternalMetricsAPI {
		externalMetricsProvider = nil
	}

	// In this example, the same provider implements both Custom Metrics API and External Metrics API
	server, err := config.Complete().New("sample-custom-metrics-adapter", customMetricsProvider, externalMetricsProvider)
	if err != nil {
		return err
	}
	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}

type SampleAdapterServerOptions struct {
	*server.CustomMetricsAdapterServerOptions

	// RemoteKubeConfigFile is the config used to list pods from the master API server
	RemoteKubeConfigFile string
	// DiscoveryInterval is the interval at which discovery information is refreshed
	DiscoveryInterval time.Duration
	// EnableCustomMetricsAPI switches on sample apiserver for Custom Metrics API
	EnableCustomMetricsAPI bool
	// EnableExternalMetricsAPI switches on sample apiserver for External Metrics API
	EnableExternalMetricsAPI bool
}
