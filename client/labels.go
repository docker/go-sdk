package client

import "maps"

const (
	// LabelBase is the base label for all Docker SDK labels.
	LabelBase = "com.docker.sdk"

	// LabelLang specifies the language.
	LabelLang = LabelBase + ".lang"

	// LabelVersion specifies the version of go-sdk's client.
	LabelVersion = LabelBase + ".client"
)

// sdkLabels is a map of labels that can be used to identify resources
// created by this library.
var sdkLabels = map[string]string{
	LabelBase:    "true",
	LabelLang:    "go",
	LabelVersion: Version(),
}

// AddSDKLabels adds the SDK labels to target.
func AddSDKLabels(target map[string]string) {
	if target == nil {
		target = make(map[string]string)
	}
	maps.Copy(target, sdkLabels)
}

// SDKLabels returns a map of labels that can be used to identify resources
// created by this library.
func SDKLabels() map[string]string {
	return map[string]string{
		LabelBase:    "true",
		LabelLang:    "go",
		LabelVersion: Version(),
	}
}
