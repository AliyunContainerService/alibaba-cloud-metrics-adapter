package util

import (
	"os"

	sls "github.com/aliyun/aliyun-log-go-sdk"
)

// Project define Project for test
var (
	ProjectName     = "test-project"
	Endpoint        = "<endpoint>"
	LogStoreName    = "<endpoint>"
	AccessKeyID     = "<accessKeyId>"
	AccessKeySecret = "<accessKeySecret>"
	Client          *sls.Client
)

func init() {
	ProjectName = os.Getenv("LOG_TEST_PROJECT")
	AccessKeyID = os.Getenv("LOG_TEST_ACCESS_KEY_ID")
	AccessKeySecret = os.Getenv("LOG_TEST_ACCESS_KEY_SECRET")
	Endpoint = os.Getenv("LOG_TEST_ENDPOINT")
	LogStoreName = os.Getenv("LOG_TEST_LOGSTORE")

	Client = &sls.Client{
		Endpoint:        Endpoint,
		AccessKeyID:     AccessKeyID,
		AccessKeySecret: AccessKeySecret,
	}
}
