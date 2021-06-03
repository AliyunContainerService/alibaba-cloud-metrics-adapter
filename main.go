package main

import (
	"flag"
	"fmt"
	kube "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/kubernetes"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus"
	naming2 "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus/naming"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/alibaba_cloud_provider"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheus_provider"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"io/ioutil"
	"k8s.io/component-base/logs"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	cfg "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/config"
	"k8s.io/client-go/transport"
	"k8s.io/klog"
)

type AlibabaMetricsAdapter struct {
	basecmd.AdapterBase
	// PrometheusURL is the URL describing how to connect to Prometheus.  Query parameters configure connection options.
	PrometheusURL string
	// PrometheusAuthInCluster enables using the auth details from the in-cluster kubeconfig to connect to Prometheus
	PrometheusAuthInCluster bool
	// PrometheusAuthConf is the kubeconfig file that contains auth details used to connect to Prometheus
	PrometheusAuthConf string
	// PrometheusCAFile points to the file containing the ca-root for connecting with Prometheus
	PrometheusCAFile string
	// PrometheusTokenFile points to the file that contains the bearer token when connecting with Prometheus
	PrometheusTokenFile string
	// AdapterConfigFile points to the file containing the metrics discovery configuration.
	AdapterConfigFile string
	// MetricsRelistInterval is the interval at which to relist the set of available metrics
	MetricsRelistInterval time.Duration
	// MetricsMaxAge is the period to query available metrics for
	MetricsMaxAge time.Duration

	metricsConfig *cfg.MetricsDiscoveryConfig
}

func makeAlibabaCloudProvider(cmd *AlibabaMetricsAdapter) (provider.ExternalMetricsProvider, error) {
	if cmd.PrometheusURL == "" {
		klog.Infof("prometheus url is empty")
	}

	alibabaProvider := alibaba_cloud_provider.NewAlibabaCloudProvider(cmd.PrometheusURL)

	// TODO custom metrics will be supported later after multi custom adapter support.
	//cmd.WithCustomMetrics(metricProvider)
	return alibabaProvider, nil
}

func makePrometheusProvider(cmd *AlibabaMetricsAdapter, stopCh <-chan struct{}) (provider.CustomMetricsProvider, error) {
	if len(cmd.metricsConfig.Rules) == 0 {
		return nil, nil
	}

	if cmd.MetricsMaxAge < cmd.MetricsRelistInterval {
		return nil, fmt.Errorf("max age must not be less than relist interval")
	}

	// grab the mapper and dynamic client
	mapper, err := cmd.RESTMapper()
	if err != nil {
		return nil, fmt.Errorf("unable to construct RESTMapper: %v", err)
	}
	dynClient, err := cmd.DynamicClient()
	if err != nil {
		return nil, fmt.Errorf("unable to construct Kubernetes client: %v", err)
	}

	// extract the namers
	namers, err := naming2.NamersFromConfig(cmd.metricsConfig.Rules, mapper)
	if err != nil {
		return nil, fmt.Errorf("unable to construct naming scheme from metrics rules: %v", err)
	}

	// make the prom client
	promClient, err := cmd.makePromClient()
	if err != nil {
		klog.Fatalf("unable to construct Prometheus client: %v", err)
	}

	// construct the provider and start it
	cmProvider, runner := prometheus_provider.NewPrometheusProvider(mapper, dynClient, promClient, namers, cmd.MetricsRelistInterval, cmd.MetricsMaxAge)
	runner.RunUntil(stopCh)

	return cmProvider, nil
}

func (cmd *AlibabaMetricsAdapter) addFlags() {
	cmd.Flags().StringVar(&cmd.PrometheusURL, "prometheus-url", cmd.PrometheusURL,
		"URL for connecting to Prometheus.")
	cmd.Flags().BoolVar(&cmd.PrometheusAuthInCluster, "prom-auth-incluster", cmd.PrometheusAuthInCluster,
		"use auth details from the in-cluster kubeconfig when connecting to prom.")
	cmd.Flags().StringVar(&cmd.PrometheusAuthConf, "prom-auth-config", cmd.PrometheusAuthConf,
		"kubeconfig file used to configure auth when connecting to Prometheus.")
	cmd.Flags().StringVar(&cmd.PrometheusCAFile, "prom-ca-file", cmd.PrometheusCAFile,
		"Optional CA file to use when connecting with Prometheus")
	cmd.Flags().StringVar(&cmd.PrometheusTokenFile, "prom-token-file", cmd.PrometheusTokenFile,
		"Optional file containing the bearer token to use when connecting with Prometheus")
	cmd.Flags().StringVar(&cmd.AdapterConfigFile, "config", cmd.AdapterConfigFile,
		"Configuration file containing details of how to transform between Prometheus metrics "+
			"and custom metrics API resources")
	cmd.Flags().DurationVar(&cmd.MetricsRelistInterval, "metrics-relist-interval", cmd.MetricsRelistInterval, ""+
		"interval at which to re-list the set of all available metrics from Prometheus")
	cmd.Flags().DurationVar(&cmd.MetricsMaxAge, "metrics-max-age", cmd.MetricsMaxAge, ""+
		"period for which to query the set of available metrics from Prometheus")
}

func (cmd *AlibabaMetricsAdapter) loadConfig() error {
	// load metrics discovery configuration
	if cmd.AdapterConfigFile == "" {
		return fmt.Errorf("no metrics discovery configuration file specified (make sure to use --config)")
	}

	metricsConfig, err := cfg.FromFile(cmd.AdapterConfigFile)
	if err != nil {
		return fmt.Errorf("unable to load metrics discovery configuration: %v", err)
	}

	cmd.metricsConfig = metricsConfig

	return nil
}

func (cmd *AlibabaMetricsAdapter) makePromClient() (prometheus.Client, error) {
	baseURL, err := url.Parse(cmd.PrometheusURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Prometheus URL %q: %v", baseURL, err)
	}

	var httpClient *http.Client

	if cmd.PrometheusCAFile != "" {
		prometheusCAClient, err := prometheus.MakePrometheusCAClient(cmd.PrometheusCAFile)
		if err != nil {
			return nil, err
		}
		httpClient = prometheusCAClient
		klog.Info("successfully loaded ca from file")
	} else {
		kubeconfigHTTPClient, err := kube.MakeKubeconfigHTTPClient(cmd.PrometheusAuthInCluster, cmd.PrometheusAuthConf)
		if err != nil {
			return nil, err
		}
		httpClient = kubeconfigHTTPClient
		klog.Info("successfully using in-cluster auth")
	}

	if cmd.PrometheusTokenFile != "" {
		data, err := ioutil.ReadFile(cmd.PrometheusTokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read prom-token-file: %v", err)
		}
		httpClient.Transport = transport.NewBearerAuthRoundTripper(string(data), httpClient.Transport)
	}

	genericPromClient := prometheus.NewGenericAPIClient(httpClient, baseURL)
	instrumentedGenericPromClient := prometheus.InstrumentGenericAPIClient(genericPromClient, baseURL.String())
	return prometheus.NewClientForAPI(instrumentedGenericPromClient), nil
}

func main() {
	http.HandleFunc("/reload", func(writer http.ResponseWriter, request *http.Request) {
		os.Exit(0)
	})

	logs.InitLogs()
	defer logs.FlushLogs()

	// golang 1.6 or before
	if len(os.Getenv("GOMAXPROvCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	//log.InitFlags(cl)
	cmd := &AlibabaMetricsAdapter{
		PrometheusURL:         "http://ack-prom-operator-prom.monitoring.svc:9090",
		MetricsRelistInterval: 10 * time.Minute,
		MetricsMaxAge:         20 * time.Minute,
		metricsConfig:         new(cfg.MetricsDiscoveryConfig),
	}

	cmd.addFlags()
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Flags().Parse(os.Args); err != nil {
		klog.Fatalf("unable to parse flags: %v", err)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	//
	alibabaCloudProvider, err := makeAlibabaCloudProvider(cmd)
	if err != nil {
		klog.Fatalf("unable to construct alibabCloudProvider: %v", err)
	}

	if alibabaCloudProvider != nil {
		cmd.WithExternalMetrics(alibabaCloudProvider)
	}

	// load the config
	if err := cmd.loadConfig(); err != nil {
		klog.Fatalf("unable to load metrics discovery config: %v", err)
	}

	prometheusProvider, err := makePrometheusProvider(cmd, stopCh)
	if err != nil {
		klog.Fatalf("unable to construct prometheusProvider: %v", err)
	}

	if prometheusProvider != nil {
		cmd.WithCustomMetrics(prometheusProvider)
	}

	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	if err := cmd.Run(stopCh); err != nil {
		klog.Fatalf("Failed to run alibaba-cloud-metrics-adapter: %v", err)
	}
}
