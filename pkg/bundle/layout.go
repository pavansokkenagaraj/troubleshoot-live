package bundle

// Layout defines paths under which are particular resources stored.
type Layout interface {
	ClusterInfo() string
	ClusterResources() string
	PodLogs() string
	ConfigMaps() string
	Secrets() string
}

type defaultLayout struct{}

func (defaultLayout) ClusterInfo() string {
	return "k8s/cluster-info"
}

func (defaultLayout) ClusterResources() string {
	return "k8s/cluster-resources"
}

func (defaultLayout) PodLogs() string {
	return "k8s/pod-logs"
}

func (defaultLayout) ConfigMaps() string {
	return "k8s/cluster-resources/configmaps"
}

func (defaultLayout) Secrets() string {
	return "k8s/cluster-resources/secrets"
}
