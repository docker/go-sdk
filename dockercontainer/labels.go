package dockercontainer

const (
	// LabelBase is the base label for all Docker labels.
	LabelBase = "com.docker.sdk"

	// LabelLang specifies the language which created the container.
	LabelLang = LabelBase + ".lang"

	// LabelVersion specifies the version of testcontainers which created the container.
	LabelVersion = LabelBase + ".version"
)

// SDKLabels returns a map of labels that can be used to identify resources
// created by this library.
var SDKLabels = map[string]string{
	LabelBase:    "true",
	LabelLang:    "go",
	LabelVersion: Version(),
}

// AddSDKLabels adds the SDK labels to target.
func AddSDKLabels(target map[string]string) {
	for k, v := range SDKLabels {
		target[k] = v
	}
}
