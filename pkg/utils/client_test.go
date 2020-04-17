package utils

import (
	"os"
	"testing"
)

func TestGetAccessUserInfoFromEnv(t *testing.T) {
	os.Setenv("Region", "Region")

	_, err := GetRegionFromEnv()
	if err != nil {
		t.Fatalf("Failed to GetAccessUserInfoFromEnv because of %v", err)
	}
	t.Log("pass TestGetAccessUserInfoFromEnv")
}
