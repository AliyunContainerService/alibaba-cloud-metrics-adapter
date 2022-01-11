package main

import (
	"flag"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/options"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"os"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	opts := options.NewAlibabaMetricsAdapterOptions()
	opts.AddFlags()
	opts.Flags().AddGoFlagSet(flag.CommandLine)
	if err := opts.Flags().Parse(os.Args); err != nil {
		klog.Fatalf("unable to parse flags: %v", err)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	providerManager, err := provider.NewProviderManager(opts, stopCh)
	if err != nil {
		log.Fatalf("Failed to init alibaba-cloud-metrics-adapter,because of %v", err)
	}

	// register custom metrics provider
	opts.WithCustomMetrics(providerManager)
	// register external metrics provider
	opts.WithExternalMetrics(providerManager)

	// export reload endpoint
	http.HandleFunc("/reload", func(writer http.ResponseWriter, request *http.Request) {
		os.Exit(0)
	})
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	if err := opts.Run(stopCh); err != nil {
		klog.Fatalf("Failed to run alibaba-cloud-metrics-adapter: %v", err)
	}
}

