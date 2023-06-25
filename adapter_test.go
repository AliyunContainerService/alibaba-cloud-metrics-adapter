package main

import (
	"flag"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	"testing"
)

func TestFlags(t *testing.T) {

	opts := prometheusProvider.NewAlibabaMetricsAdapterOptions()
	prometheusProvider.GlobalConfig = opts
	opts.AddFlags()
	fakeCommandLines := []string{
		"--secure-port=443",
		"--prometheus-url=https://cn-shanghai.arms.aliyuncs.com:9443/api/v1/prometheus/xxxx/1251182063904492/xxxx/cn-shanghai",
		"--config=/etc/adapter/config.yaml",
		"--prometheus-header=Authorization=xxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"--metrics-relist-interval=1m",
		"--v=9",
	}
	opts.Flags().AddGoFlagSet(flag.CommandLine)
	//opts.Flags().AddGoFlagSet(flag.NewFlagSet(fakeCommandLine, 1))
	if err := opts.Flags().Parse(fakeCommandLines); err != nil {
		t.Fatalf("unable to parse flags: %v", err)
	}

	t.Logf("finish, opts: %v", opts)
}
