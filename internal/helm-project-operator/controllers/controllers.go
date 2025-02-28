package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/k3s-io/helm-controller/pkg/controllers/chart"
	"github.com/rancher/prometheus-federator/internal/helm-locker/controllers/release"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/hardened"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/project"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/wrangler/v3/pkg/leader"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Register registers all controllers for the Helm Project Operator based on the provided options
func Register(ctx context.Context, systemNamespace string, cfg clientcmd.ClientConfig, opts common.Options) error {
	if len(systemNamespace) == 0 {
		return errors.New("cannot start controllers on system namespace: system namespace not provided")
	}
	// always add the systemNamespace to the systemNamespaces provided
	opts.SystemNamespaces = append(opts.SystemNamespaces, systemNamespace)

	// parse values.yaml and questions.yaml from file
	valuesYaml, questionsYaml, err := parseValuesAndQuestions(opts.ChartContent)
	if err != nil {
		logrus.Fatal(err)
	}

	appCtx, err := setup.NewAppContext(cfg, systemNamespace, opts)
	if err != nil {
		return err
	}

	appCtx.EventBroadcaster.StartLogging(logrus.Debugf)
	appCtx.EventBroadcaster.StartRecordingToSink(&typedv1.EventSinkImpl{
		Interface: appCtx.K8s.CoreV1().Events(systemNamespace),
	})
	recorder := appCtx.EventBroadcaster.NewRecorder(schemes.All, corev1.EventSource{
		Component: "helm-project-operator",
		Host:      opts.NodeName,
	})

	if !opts.DisableHardening {
		hardeningOpts, err := common.LoadHardeningOptionsFromFile(opts.HardeningOptionsFile)
		if err != nil {
			return err
		}
		hardened.Register(ctx,
			appCtx.Apply,
			hardeningOpts,
			// watches
			appCtx.Core.Namespace(),
			appCtx.Core.Namespace().Cache(),
			// generates
			appCtx.Core.ServiceAccount(),
			appCtx.Networking.NetworkPolicy(),
		)
	}

	logrus.Debug("Registering namespace controller...")
	projectGetter := namespace.Register(ctx,
		appCtx.Apply,
		systemNamespace,
		valuesYaml,
		questionsYaml,
		opts,
		// watches and generates
		appCtx.Core.Namespace(),
		appCtx.Core.Namespace().Cache(),
		appCtx.Core.ConfigMap(),
		// enqueues
		appCtx.ProjectHelmChart(),
		appCtx.ProjectHelmChart().Cache(),
		appCtx.Dynamic,
	)

	if len(opts.ControllerName) == 0 {
		opts.ControllerName = "helm-project-operator"
	}

	valuesOverride, err := common.LoadValuesOverrideFromFile(opts.ValuesOverrideFile)
	if err != nil {
		return err
	}
	logrus.Infof("Registering Project Controller...")
	project.Register(ctx,
		systemNamespace,
		opts,
		valuesOverride,
		appCtx.Apply,
		// watches
		appCtx.ProjectHelmChart(),
		appCtx.ProjectHelmChart().Cache(),
		appCtx.Core.ConfigMap(),
		appCtx.Core.ConfigMap().Cache(),
		appCtx.RBAC.Role(),
		appCtx.RBAC.Role().Cache(),
		appCtx.RBAC.ClusterRoleBinding(),
		appCtx.RBAC.ClusterRoleBinding().Cache(),
		// watches and generates
		appCtx.HelmController.HelmChart(),
		appCtx.HelmLocker.HelmRelease(),
		appCtx.Core.Namespace(),
		appCtx.Core.Namespace().Cache(),
		appCtx.RBAC.RoleBinding(),
		appCtx.RBAC.RoleBinding().Cache(),
		projectGetter,
	)

	if !opts.DisableEmbeddedHelmLocker {
		logrus.Infof("Registering embedded Helm Locker...")
		release.Register(ctx,
			systemNamespace,
			opts.ControllerName,
			appCtx.HelmLocker.HelmRelease(),
			appCtx.HelmLocker.HelmRelease().Cache(),
			appCtx.Core.Secret(),
			appCtx.Core.Secret().Cache(),
			appCtx.K8s,
			appCtx.ObjectSetRegister,
			appCtx.ObjectSetHandler,
			recorder,
		)
	}

	if !opts.DisableEmbeddedHelmController {
		logrus.Infof("Registering embedded Helm Controller...")
		chart.Register(ctx,
			systemNamespace,
			opts.ControllerName,
			// this has to be cluster-admin for k3s reasons
			"cluster-admin",
			"6443",
			appCtx.K8s,
			appCtx.Apply,
			recorder,
			appCtx.HelmController.HelmChart(),
			appCtx.HelmController.HelmChart().Cache(),
			appCtx.HelmController.HelmChartConfig(),
			appCtx.HelmController.HelmChartConfig().Cache(),
			appCtx.Batch.Job(),
			appCtx.Batch.Job().Cache(),
			appCtx.RBAC.ClusterRoleBinding(),
			appCtx.Core.ServiceAccount(),
			appCtx.Core.ConfigMap(),
			appCtx.Core.Secret(),
		)
	}

	leader.RunOrDie(ctx, systemNamespace, fmt.Sprintf("helm-project-operator-%s-lock", opts.ReleaseName), appCtx.K8s, func(ctx context.Context) {
		if err := appCtx.Start(ctx); err != nil {
			logrus.Fatal(err)
		}
		logrus.Info("All controllers have been started")
	})

	return nil
}
