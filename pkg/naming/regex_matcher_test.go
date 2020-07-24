package naming

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/config"
)

func TestReMatcherIs(t *testing.T) {
	filter := config.RegexFilter{
		Is: "my_.*",
	}

	matcher, err := NewReMatcher(filter)
	require.NoError(t, err)

	result := matcher.Matches("my_label")
	require.True(t, result)

	result = matcher.Matches("your_label")
	require.False(t, result)
}

func TestReMatcherIsNot(t *testing.T) {
	filter := config.RegexFilter{
		IsNot: "my_.*",
	}

	matcher, err := NewReMatcher(filter)
	require.NoError(t, err)

	result := matcher.Matches("my_label")
	require.False(t, result)

	result = matcher.Matches("your_label")
	require.True(t, result)
}

func TestEnforcesIsOrIsNotButNotBoth(t *testing.T) {
	filter := config.RegexFilter{
		Is:    "my_.*",
		IsNot: "your_.*",
	}

	_, err := NewReMatcher(filter)
	require.Error(t, err)
}
