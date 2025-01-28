package releases

import (
	"sync"

	"github.com/sirupsen/logrus"
	rspb "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/client-go/kubernetes"
)

type HelmReleaseGetter interface {
	Last(namespace, name string) (*rspb.Release, error)
}

func NewHelmReleaseGetter(k8s kubernetes.Interface) HelmReleaseGetter {
	return &latestReleaseGetter{
		K8s:               k8s,
		namespacedStorage: make(map[string]*storage.Storage),
	}
}

type latestReleaseGetter struct {
	K8s kubernetes.Interface

	namespacedStorage map[string]*storage.Storage
	storageLock       sync.Mutex
}

func (g *latestReleaseGetter) getStore(namespace string) *storage.Storage {
	g.storageLock.Lock()
	defer g.storageLock.Unlock()
	store, ok := g.namespacedStorage[namespace]
	if ok && store != nil {
		return store
	}
	store = storage.Init(driver.NewSecrets(g.K8s.CoreV1().Secrets(namespace)))
	store.Log = logrus.Debugf
	g.namespacedStorage[namespace] = store
	return store
}

func (g *latestReleaseGetter) Last(namespace, name string) (*rspb.Release, error) {
	store := g.getStore(namespace)
	return store.Last(name)
}
