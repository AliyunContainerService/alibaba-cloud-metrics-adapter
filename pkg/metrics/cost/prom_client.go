package cost

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/options"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
	"k8s.io/klog/v2"
	"net/http"
	"net/url"
	prom "sigs.k8s.io/prometheus-adapter/pkg/client"
	"strings"
)

func (cs *COSTMetricSource) getPrometheusClient() (prom.Client, error) {
	PrometheusURL := options.GlobalConfig.PrometheusURL
	PrometheusCAFile := options.GlobalConfig.PrometheusCAFile
	PrometheusAuthInCluster := options.GlobalConfig.PrometheusAuthInCluster
	PrometheusAuthConf := options.GlobalConfig.PrometheusAuthConf
	PrometheusTokenFile := options.GlobalConfig.PrometheusTokenFile
	PrometheusHeaders := options.GlobalConfig.PrometheusHeaders
	baseURL, err := url.Parse(PrometheusURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Prometheus URL %q: %v", baseURL, err)
	}

	var httpClient *http.Client

	if PrometheusCAFile != "" {
		prometheusCAClient, err := cs.makePrometheusCAClient(PrometheusCAFile)
		if err != nil {
			return nil, err
		}
		httpClient = prometheusCAClient
		klog.Info("successfully loaded ca from file")
	} else {
		kubeconfigHTTPClient, err := cs.makeKubeconfigHTTPClient(PrometheusAuthInCluster, PrometheusAuthConf)
		if err != nil {
			return nil, err
		}
		httpClient = kubeconfigHTTPClient
		klog.Info("successfully using in-cluster auth")
	}

	if PrometheusTokenFile != "" {
		data, err := ioutil.ReadFile(PrometheusTokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read prometheus-token-file: %v", err)
		}
		httpClient.Transport = transport.NewBearerAuthRoundTripper(string(data), httpClient.Transport)
	}

	genericPromClient := prom.NewGenericAPIClient(httpClient, baseURL, cs.parseHeaderArgs(PrometheusHeaders))
	instrumentedGenericPromClient := utils.InstrumentGenericAPIClient(genericPromClient, baseURL.String())
	return prom.NewClientForAPI(instrumentedGenericPromClient), nil

}

func (cs *COSTMetricSource) makePrometheusCAClient(caFilename string) (*http.Client, error) {
	data, err := ioutil.ReadFile(caFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to read prometheus-ca-file: %v", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no certs found in prometheus-ca-file")
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}, nil
}

// makeKubeconfigHTTPClient constructs an HTTP for connecting with the given auth options.
func (cs *COSTMetricSource) makeKubeconfigHTTPClient(inClusterAuth bool, kubeConfigPath string) (*http.Client, error) {
	// make sure we're not trying to use two different sources of auth
	if inClusterAuth && kubeConfigPath != "" {
		return nil, fmt.Errorf("may not use both in-cluster auth and an explicit kubeconfig at the same time")
	}

	// return the default client if we're using no auth
	if !inClusterAuth && kubeConfigPath == "" {
		return http.DefaultClient, nil
	}

	var authConf *rest.Config
	if kubeConfigPath != "" {
		var err error
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		authConf, err = loader.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to construct  auth configuration from %q for connecting to Prometheus: %v", kubeConfigPath, err)
		}
	} else {
		var err error
		authConf, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to construct in-cluster auth configuration for connecting to Prometheus: %v", err)
		}
	}
	tr, err := rest.TransportFor(authConf)
	if err != nil {
		return nil, fmt.Errorf("unable to construct client transport for connecting to Prometheus: %v", err)
	}
	return &http.Client{Transport: tr}, nil
}

func (cs *COSTMetricSource) parseHeaderArgs(args []string) http.Header {
	headers := make(http.Header, len(args))
	for _, h := range args {
		parts := strings.SplitN(h, "=", 2)
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		headers.Add(parts[0], value)
	}
	return headers
}
