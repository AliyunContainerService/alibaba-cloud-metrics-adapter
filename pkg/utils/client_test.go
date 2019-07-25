package utils

import (
	"os"
	"testing"
)

func TestGetAccessUserInfoFromMeta(t *testing.T) {
	t.Skip("skip TestGetAccessUserInfoFromMeta in unittest")
	_, err := GetAccessUserInfoFromMeta()
	if err != nil {
		t.Fatal("Failed to GetAccessUserInfoFromMeta")
	}
	t.Log("pass TestGetAccessUserInfoFromMeta")
}

func TestGetAccessUserInfoFromEnv(t *testing.T) {
	os.Setenv("AccessKeyId", "AccessKeyId")
	os.Setenv("AccessKeySecret", "AccessKeySecret")
	os.Setenv("Region", "Region")

	_, err := GetAccessUserInfoFromEnv()
	if err != nil {
		t.Fatalf("Failed to GetAccessUserInfoFromEnv because of %v", err)
	}
	t.Log("pass TestGetAccessUserInfoFromEnv")
}
