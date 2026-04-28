package istio

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Istiod struct {
	pulumi.ResourceState

	Release *helm.Release
}

type IstiodArgs struct {
	// Version of Istiod chart (default: 1.28.2)
	Version string
	// Optional profile to use (for example: ambient, demo, preview, stable, remote).
	// Leave empty to use the chart defaults.
	Profile string
}

func NewIstiod(ctx *pulumi.Context, name string, args *IstiodArgs, opts ...pulumi.ResourceOption) (*Istiod, error) {
	component := &Istiod{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("istio:Istiod", name, component, opts...)
	if err != nil {
		return nil, err
	}

	version := DefaultVersion
	profile := ""
	if args != nil {
		if args.Version != "" {
			version = args.Version
		}
		if args.Profile != "" {
			profile = args.Profile
		}
	}

	releaseArgs := &helm.ReleaseArgs{
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String(IstioHelmRepo),
		},
		Name:            pulumi.String("istio-istiod"),
		Namespace:       pulumi.String("istio-system"),
		Chart:           pulumi.String("istiod"),
		CreateNamespace: pulumi.BoolPtr(true),
		Version:         pulumi.String(version),
	}

	if profile != "" {
		releaseArgs.Values = pulumi.Map{
			"profile": pulumi.String(profile),
		}
	}

	release, err := helm.NewRelease(ctx, ns.Get(), releaseArgs, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Release = release

	return component, nil
}
