package inference

// ExperimentalEndpointsPrefix is used to prefix all <paths.InferencePrefix> routes on the Docker
// socket while they are still in their experimental stage. This prefix doesn't
// apply to endpoints on model-runner.docker.internal.
const ExperimentalEndpointsPrefix = "/exp/vDD4.40"

// InferencePrefix is the prefix for inference related related routes.
var InferencePrefix = "/engines"

// ModelsPrefix is the prefix for all model manager related routes.
var ModelsPrefix = "/models"
