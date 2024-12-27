package release

import (
	v1alpha1 "github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	rspb "helm.sh/helm/v3/pkg/release"
)

func newReleaseInfo(release *rspb.Release) *releaseInfo {
	info := &releaseInfo{}
	info.Version = int(release.Version)
	info.Manifest = release.Manifest
	if release.Info != nil {
		info.Description = release.Info.Description
		info.Notes = release.Info.Notes
		switch release.Info.Status {
		case rspb.StatusUnknown:
			info.State = v1alpha1.UnknownState
		case rspb.StatusDeployed:
			info.State = v1alpha1.DeployedState
		case rspb.StatusUninstalled:
			info.State = v1alpha1.UninstalledState
		case rspb.StatusSuperseded:
			// note: this should never be the case since we always get the latest secret
			info.State = v1alpha1.ErrorState
		case rspb.StatusFailed:
			info.State = v1alpha1.FailedState
		default:
			// uninstalling, pending install, pending upgrade, pending rollback
			info.State = v1alpha1.TransitioningState
		}
	}
	return info
}

type releaseInfo struct {
	Version     int
	Manifest    string
	Description string
	Notes       string
	State       string
}

func (i *releaseInfo) Locked() bool {
	return i.State == v1alpha1.DeployedState
}

func (i *releaseInfo) GetUpdatedStatus(helmRelease *v1alpha1.HelmRelease) *v1alpha1.HelmRelease {
	helmRelease.Status.Version = i.Version
	helmRelease.Status.Description = i.Description
	helmRelease.Status.State = i.State
	helmRelease.Status.Notes = i.Notes
	return helmRelease
}
