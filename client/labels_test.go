package client_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
)

func TestAddSDKLabels(t *testing.T) {
	labels := map[string]string{}

	client.AddSDKLabels(labels)
	require.Contains(t, labels, client.LabelBase)
	require.Contains(t, labels, client.LabelLang)
	require.Contains(t, labels, client.LabelVersion)

	t.Run("idempotent", func(t *testing.T) {
		sdkLabels := client.SDKLabels()
		sdkLabels["foo"] = "bar"

		labels := make(map[string]string)
		client.AddSDKLabels(labels)
		require.NotEqual(t, sdkLabels, labels)
		require.NotContains(t, labels, "foo")
	})
}
