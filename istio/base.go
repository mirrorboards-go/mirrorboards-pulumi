package istio

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Base struct {
	pulumi.ResourceState

	Release *helm.Release
}

type BaseArgs struct {
	// Version of Istio base chart (default: 1.28.2)
	Version string
}

func NewBase(ctx *pulumi.Context, name string, args *BaseArgs, opts ...pulumi.ResourceOption) (*Base, error) {
	component := &Base{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("istio:Base", name, component, opts...)
	if err != nil {
		return nil, err
	}

	version := DefaultVersion
	if args != nil && args.Version != "" {
		version = args.Version
	}

	release, err := helm.NewRelease(ctx, ns.Get(), &helm.ReleaseArgs{
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String(IstioHelmRepo),
		},
		Name:            pulumi.String("istio-base"),
		Namespace:       pulumi.String("istio-system"),
		Chart:           pulumi.String("base"),
		CreateNamespace: pulumi.BoolPtr(true),
		Version:         pulumi.String(version),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Release = release

	return component, nil
}
