package istio

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Kiali struct {
	pulumi.ResourceState

	Release *helm.Release
}

type KialiArgs struct {
	// Version of Kiali chart (optional, uses latest if not specified)
	Version string
	// AuthStrategy for Kiali (default: anonymous)
	AuthStrategy string
}

func NewKiali(ctx *pulumi.Context, name string, args *KialiArgs, opts ...pulumi.ResourceOption) (*Kiali, error) {
	component := &Kiali{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("istio:Kiali", name, component, opts...)
	if err != nil {
		return nil, err
	}

	authStrategy := "anonymous"
	if args != nil && args.AuthStrategy != "" {
		authStrategy = args.AuthStrategy
	}

	releaseArgs := &helm.ReleaseArgs{
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://kiali.org/helm-charts"),
		},
		Namespace: pulumi.String("istio-system"),
		Chart:     pulumi.String("kiali-server"),
		Values: pulumi.Map{
			"auth": pulumi.Map{
				"strategy": pulumi.String(authStrategy),
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
