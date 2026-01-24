package istio

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CNI struct {
	pulumi.ResourceState

	Release *helm.Release
}

type CNIArgs struct {
	// Version of Istio CNI chart (default: 1.28.2)
	Version string
	// Profile to use (default: ambient)
	Profile string
}

func NewCNI(ctx *pulumi.Context, name string, args *CNIArgs, opts ...pulumi.ResourceOption) (*CNI, error) {
	component := &CNI{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("istio:CNI", name, component, opts...)
	if err != nil {
		return nil, err
	}

	version := DefaultVersion
	profile := "ambient"
	if args != nil {
		if args.Version != "" {
			version = args.Version
		}
		if args.Profile != "" {
			profile = args.Profile
		}
	}

	release, err := helm.NewRelease(ctx, ns.Get(), &helm.ReleaseArgs{
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String(IstioHelmRepo),
		},
		Name:            pulumi.String("istio-cni"),
		Namespace:       pulumi.String("istio-system"),
		Chart:           pulumi.String("cni"),
		CreateNamespace: pulumi.BoolPtr(true),
		Version:         pulumi.String(version),
		Values: pulumi.Map{
			"profile": pulumi.String(profile),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Release = release

	return component, nil
}
