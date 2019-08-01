package sls

import (
	"testing"

	"encoding/json"

	"github.com/golang/glog"
	"github.com/stretchr/testify/suite"
)

func TestETLJob_UnmarshalJSON(t *testing.T) {
	suite.Run(t, new(ETLJobTestSuite))
	glog.Flush()
}

type ETLJobTestSuite struct {
	suite.Suite
}

func (s *ETLJobTestSuite) SetupTest() {

}

func (s *ETLJobTestSuite) TestUnmarshalJSON_JSON_Param() {
	job := ETLJob{}
	str := `
{
 "etlJobName": "b8be831fac391d65b709e9a4f663e559eaa31e5a",
 "sourceConfig": {
  "logstoreName": "etl-log"
 },
 "triggerConfig": {
  "maxRetryTime": 3,
  "triggerInterval": 60,
  "roleArn": "acs:ram::12345:role/invoke-all"
 },
 "functionConfig": {
  "functionProvider": "FunctionCompute",
  "endpoint": "https://cn-hangzhou-internal.fc.aliyuncs.com",
  "accountId": "12345",
  "regionName": "cn-hangzhou",
  "serviceName": "demo",
  "functionName": "helloworld"
 },
 "functionParameter": {
  "a": "b"
 },
 "logConfig": {
  "endpoint": "cn-shanghai.log.aliyuncs.com",
  "projectName": "ali-fc-test",
  "logstoreName": "test"
 },
 "enable": true,
 "createTime": 1506469441,
 "updateTime": 1506469441
}
	`
	err := json.Unmarshal([]byte(str), &job)
	s.Nil(err)
	s.Equal("b", job.FunctionParameter.(map[string]interface{})["a"])
}

func (s *ETLJobTestSuite) TestUnmarshalJSON_String_Param() {
	job := ETLJob{}
	str := `
{
 "etlJobName": "b8be831fac391d65b709e9a4f663e559eaa31e5a",
 "sourceConfig": {
  "logstoreName": "etl-log"
 },
 "triggerConfig": {
  "maxRetryTime": 3,
  "triggerInterval": 60,
  "roleArn": "acs:ram::12345:role/invoke-all"
 },
 "functionConfig": {
  "functionProvider": "FunctionCompute",
  "endpoint": "https://cn-hangzhou-internal.fc.aliyuncs.com",
  "accountId": "12345",
  "regionName": "cn-hangzhou",
  "serviceName": "demo",
  "functionName": "helloworld"
 },
 "functionParameter": "{\"a\": \"b\"}",
 "logConfig": {
  "endpoint": "cn-shanghai.log.aliyuncs.com",
  "projectName": "ali-fc-test",
  "logstoreName": "test"
 },
 "enable": true,
 "createTime": 1506469441,
 "updateTime": 1506469441
}
	`
	err := json.Unmarshal([]byte(str), &job)
	s.Nil(err)
	s.Equal("b", job.FunctionParameter.(map[string]interface{})["a"])
}
