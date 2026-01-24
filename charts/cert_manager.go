package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CertManager struct {
	pulumi.ResourceState

	Release *helm.Release
}

type CertManagerArgs struct {
	// Version of cert-manager chart (optional, uses latest if not specified)
	Version string
}

func NewCertManager(ctx *pulumi.Context, name string, args *CertManagerArgs, opts ...pulumi.ResourceOption) (*CertManager, error) {
	component := &CertManager{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("charts:CertManager", name, component, opts...)
	if err != nil {
		return nil, err
	}

	releaseArgs := &helm.ReleaseArgs{
		Chart:       pulumi.String("cert-manager"),
		ReuseValues: pulumi.Bool(true),
		WaitForJobs: pulumi.BoolPtr(true),
		Namespace:   pulumi.String("kube-system"),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.jetstack.io"),
		},
		Values: pulumi.Map{
			"config": pulumi.Map{
				"enableGatewayAPI": pulumi.Bool(true),
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
