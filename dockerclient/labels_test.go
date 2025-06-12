package dockerclient_test

import (
	"testing"

	"github.com/docker/go-sdk/dockerclient"
	"github.com/stretchr/testify/require"
)

func TestAddSDKLabels(t *testing.T) {
	labels := map[string]string{}

	dockerclient.AddSDKLabels(labels)
	require.Contains(t, labels, dockerclient.LabelBase)
	require.Contains(t, labels, dockerclient.LabelLang)
	require.Contains(t, labels, dockerclient.LabelVersion)
}
