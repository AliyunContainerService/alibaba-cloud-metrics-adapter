package utils

import (
	"errors"
	"os"
)

func GetRegionFromEnv()(region string,err error) {
	region = os.Getenv("Region")
	if region==""{
		return "",errors.New("not found region info in env")
	}
	return  region,nil
}
