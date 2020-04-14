package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/denverdino/aliyungo/metadata"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"time"
)

const (
	ConfigPath = "/var/addon/token-config"
)

type AccessUserInfo struct {
	AccessKeyId     string `json:"access.key.id"`
	AccessKeySecret string `json:"access.key.secret"`
	Token           string `json:"security.token"`
	Expiration      string `json:"expiration"`
	Keyring         string `json:"keyring"`
	Region          string
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func Decrypt(s string, keyring []byte) ([]byte, error) {
	cdata, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		klog.Errorf("failed to decode base64 string, err: %v", err)
		return nil, err
	}
	block, err := aes.NewCipher(keyring)
	if err != nil {
		klog.Errorf("failed to new cipher, err: %v", err)
		return nil, err
	}
	blockSize := block.BlockSize()

	iv := cdata[:blockSize]
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(cdata)-blockSize)

	blockMode.CryptBlocks(origData, cdata[blockSize:])

	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func GetAccessUserInfo() (accessUserInfo *AccessUserInfo, err error) {
	m := metadata.NewMetaData(nil)
	region, err := GetRegionFromEnv()
	if err != nil {
		region, err = m.Region()
		if err != nil {
			klog.Errorf("failed to get Region,because of %s", err.Error())
			return nil, err
		}
	}

	var akInfo AccessUserInfo
	if _, err := os.Stat(ConfigPath); err == nil {
		//获取token config json
		encodeTokenCfg, err := ioutil.ReadFile(ConfigPath)
		if err != nil {
			klog.Fatalf("failed to read token config, err: %v", err)
		}
		err = json.Unmarshal(encodeTokenCfg, &akInfo)
		if err != nil {
			klog.Fatalf("error unmarshal token config: %v", err)
		}
		keyring := akInfo.Keyring
		ak, err := Decrypt(akInfo.AccessKeyId, []byte(keyring))
		if err != nil {
			klog.Fatalf("failed to decode ak, err: %v", err)
		}

		sk, err := Decrypt(akInfo.AccessKeySecret, []byte(keyring))
		if err != nil {
			klog.Fatalf("failed to decode sk, err: %v", err)
		}

		token, err := Decrypt(akInfo.Token, []byte(keyring))
		if err != nil {
			klog.Fatalf("failed to decode token, err: %v", err)
		}
		layout := "2006-01-02T15:04:05Z"
		t, err := time.Parse(layout, akInfo.Expiration)
		if err != nil {
			fmt.Errorf(err.Error())
		}
		if t.Before(time.Now()) {
			klog.Errorf("invalid token which is expired")
		}
		akInfo.AccessKeyId = string(ak)
		akInfo.AccessKeySecret = string(sk)
		akInfo.Token = string(token)
		akInfo.Region = region
	} else {
		//兼容老的metaserver获取形式
		roleName, err := m.RoleName()
		if err != nil {
			klog.Errorf("failed to get RoleName,because of %s", err.Error())
			return nil, err
		}

		auth, err := m.RamRoleToken(roleName)
		if err != nil {
			klog.Errorf("failed to get RamRoleToken,because of %s", err.Error())
			return nil, err
		}
		akInfo.AccessKeyId = auth.AccessKeyId
		akInfo.AccessKeySecret = auth.AccessKeySecret
		akInfo.Token = auth.SecurityToken
		akInfo.Region = region
	}
	return &akInfo, nil
}
