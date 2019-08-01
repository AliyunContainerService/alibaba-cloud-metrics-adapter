package sls

import (
	"os"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/suite"
)

func TestProject(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
	glog.Flush()
}

type ProjectTestSuite struct {
	suite.Suite
	endpoint        string
	accessKeyID     string
	accessKeySecret string
	client          Client
}

func (s *ProjectTestSuite) SetupTest() {
	s.endpoint = os.Getenv("LOG_TEST_ENDPOINT")
	s.accessKeyID = os.Getenv("LOG_TEST_ACCESS_KEY_ID")
	s.accessKeySecret = os.Getenv("LOG_TEST_ACCESS_KEY_SECRET")
	s.client = Client{
		Endpoint:        s.endpoint,
		AccessKeyID:     s.accessKeyID,
		AccessKeySecret: s.accessKeySecret,
		SecurityToken:   "",
	}
}

func (s *ProjectTestSuite) TestCheckProjectExist() {
	projectName := os.Getenv("LOG_TEST_PROJECT")
	exist, err := s.client.CheckProjectExist(projectName)
	s.Nil(err)
	s.True(exist)
}

func (s *ProjectTestSuite) TestParseEndpoint() {
	assert := s.Require()

	projectName := "my-project"
	prj, err := NewLogProject(projectName, "127.0.0.1", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("http://my-project.127.0.0.1", prj.baseURL)

	prj, err = NewLogProject(projectName, "http://127.0.0.1", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("http://my-project.127.0.0.1", prj.baseURL)

	prj, err = NewLogProject(projectName, "http://127.0.0.1:8080", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("http://my-project.127.0.0.1:8080", prj.baseURL)

	prj, err = NewLogProject(projectName, "log.aliyun.com", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("http://my-project.log.aliyun.com", prj.baseURL)

	prj, err = NewLogProject(projectName, "http://log.aliyun.com", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("http://my-project.log.aliyun.com", prj.baseURL)

	prj, err = NewLogProject(projectName, "http://log.aliyun.com:8000", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("http://my-project.log.aliyun.com:8000", prj.baseURL)

	prj, err = NewLogProject(projectName, "https://log.aliyun.com:8000", "id", "key")
	assert.Nil(err)
	assert.NotNil(prj)
	assert.Equal("https://my-project.log.aliyun.com:8000", prj.baseURL)
}

func (s *ProjectTestSuite) TestUpdateProject() {
	projectName := os.Getenv("LOG_TEST_PROJECT")
	_, err := s.client.UpdateProject(projectName, "aliyun log go sdk test.")
	s.Nil(err)
}
