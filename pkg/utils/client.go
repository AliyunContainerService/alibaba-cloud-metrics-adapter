package utils

import (
	"errors"
	"github.com/denverdino/aliyungo/metadata"
	log "k8s.io/klog"
	"os"
)

type AccessUserInfo struct {
	AccessKeyId     string
	AccessKeySecret string
	Token           string
	Region          string
}

// get sts token from metadata server
func GetAccessUserInfoFromMeta() (info *AccessUserInfo, err error) {
	m := metadata.NewMetaData(nil)

	r, err := m.RoleName()
	if err != nil {
		log.Errorf("Failed to find ram role of meta, because of %v", err)
		return nil, err
	}

	auth, err := m.RamRoleToken(r)
	if err != nil {
		log.Errorf("Failed to find sts token from ecs, because of %v", err)
		return nil, err
	}

	region, err := m.Region()
	if err != nil {
		log.Errorf("Failed to get region from ecs, because of %v", err)
		return nil, err
	}

	info = &AccessUserInfo{}
	info.AccessKeyId = auth.AccessKeyId
	info.AccessKeySecret = auth.AccessKeySecret
	info.Token = auth.SecurityToken
	info.Region = region
	return info, nil
}

// get access key and secret from env
func GetAccessUserInfoFromEnv() (info *AccessUserInfo, err error) {

	err = errors.New("AccessUserInfoNotFound")

	info = &AccessUserInfo{}

	if info.AccessKeyId = os.Getenv("AccessKeyId"); info.AccessKeyId == "" {
		return
	}

	if info.AccessKeySecret = os.Getenv("AccessKeySecret"); info.AccessKeySecret == "" {
		return
	}

	if info.Region = os.Getenv("Region"); info.Region == "" {
		return
	}
	return info, nil
}

func GetAccessUserInfo() (accessUserInfo *AccessUserInfo, err error) {
	accessUserInfo, err = GetAccessUserInfoFromMeta()
	if err != nil {
		log.Warningf("Failed to get accessUserInfo from metadata server,because of %v", err)
		accessUserInfo, err = GetAccessUserInfoFromEnv()
		if err != nil {
			log.Errorf("Failed to get accessUserInfo from env,because of %v.You can provide AccessKeyId,AccessKeySecret and Region in Env", err)
		}
	}
	return accessUserInfo, err
}
