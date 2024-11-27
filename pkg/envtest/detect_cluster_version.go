package envtest

import (
	"gopkg.in/yaml.v3"
	"path/filepath"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/version"
	versions "sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"

	"github.com/mhrabovcin/troubleshoot-live/pkg/bundle"
)

// clientVersion:
//
//	buildDate: "2023-04-12T12:13:53Z"
//	compiler: gc
//	gitCommit: f89670c3aa4059d6999cb42e23ccb4f0b9a03979
//	gitTreeState: clean
//	gitVersion: v1.26.4
//	goVersion: go1.19.8
//	major: "1"
//	minor: "26"
//	platform: linux/amd64
//
// kustomizeVersion: v4.5.7
// serverVersion:
//
//	buildDate: "2024-03-14T00:54:27Z"
//	compiler: gc
//	gitCommit: 1649f592f1909b97aa3c2a0a8f968a3fd05a7b8b
//	gitTreeState: clean
//	gitVersion: v1.26.15
//	goVersion: go1.21.8
//	major: "1"
//	minor: "26"
//	platform: linux/amd64
type ClusterInfo struct {
	ServerVersion    version.Info `yaml:"serverVersion"`
	ClientVersion    version.Info `yaml:"clientVersion"`
	KustomizeVersion string       `yaml:"kustomizeVersion"`
}

func selectorFromSemver(sv *semver.Version) versions.Selector {
	// return versions.Concrete{
	// 	Major: int(sv.Major()),
	// 	Minor: int(sv.Minor()),
	// 	Patch: int(sv.Patch()),
	// }
	// default storage bucket does not contain all versions
	//
	return versions.PatchSelector{
		Major: int(sv.Major()),
		Minor: int(sv.Minor()),
		Patch: versions.AnyPoint,
	}
}

// DetectK8sVersion attempts to load k8s server version from which was bundle
// collected.
func DetectK8sVersion(b bundle.Bundle) (versions.Selector, error) {
	data, err := afero.ReadFile(b, filepath.Join(b.Layout().ClusterInfo(), "cluster-version.yaml"))
	if err != nil {
		return nil, err
	}

	var i ClusterInfo
	if err := yaml.Unmarshal(data, &i); err != nil {
		return nil, err
	}
	versionString := i.ServerVersion.String()

	if sv, err := semver.NewVersion(versionString); err == nil {
		return selectorFromSemver(sv), nil
	}

	if sv, err := semver.NewVersion(versionString); err == nil {
		return selectorFromSemver(sv), nil
	}

	major, _ := strconv.Atoi(i.ServerVersion.Major)
	minor, _ := strconv.Atoi(i.ServerVersion.Minor)
	return versions.PatchSelector{
		Major: major,
		Minor: minor,
		Patch: versions.AnyPoint,
	}, nil
}
