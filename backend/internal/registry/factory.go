package registry

// Provider identifies registry backend type for future extensibility.
type Provider string

const (
	ProviderDockerV2 Provider = "docker_v2"
	// Future: harbor, gitlab, nexus, ecr, gar, acr
)

// NewClient creates a registry client for the given provider.
// v1 only implements Docker Registry HTTP API V2.
func NewClient(provider Provider, cfg Config) (Client, error) {
	switch provider {
	case ProviderDockerV2, "":
		return NewV2Client(cfg), nil
	default:
		return NewV2Client(cfg), nil
	}
}
