package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type MetricsServer struct {
	pulumi.ResourceState

	Release *helm.Release
}

type MetricsServerArgs struct {
	// Version of metrics-server chart (optional, uses latest if not specified)
	Version string
}

func NewMetricsServer(ctx *pulumi.Context, name string, args *MetricsServerArgs, opts ...pulumi.ResourceOption) (*MetricsServer, error) {
	component := &MetricsServer{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("charts:MetricsServer", name, component, opts...)
	if err != nil {
		return nil, err
	}

	releaseArgs := &helm.ReleaseArgs{
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://kubernetes-sigs.github.io/metrics-server/"),
		},
		Name:            pulumi.String("metrics-server"),
		Namespace:       pulumi.String("kube-system"),
		Chart:           pulumi.String("metrics-server"),
		Atomic:          pulumi.BoolPtr(true),
		CreateNamespace: pulumi.BoolPtr(true),
		Values: pulumi.Map{
			"metrics": pulumi.Map{
				"enabled": pulumi.Bool(true),
			},
		},
	}

	if args != nil && args.Version != "" {
		releaseArgs.Version = pulumi.String(args.Version)
	}

	release, err := helm.NewRelease(ctx, ns.Get(), releaseArgs, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Release = release

	return component, nil
}
