package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ExternalSecrets struct {
	pulumi.ResourceState

	Release *helm.Release
}

type ExternalSecretsArgs struct {
	// Version of external-secrets chart (optional, uses latest if not specified)
	Version string
}

func NewExternalSecrets(ctx *pulumi.Context, name string, args *ExternalSecretsArgs, opts ...pulumi.ResourceOption) (*ExternalSecrets, error) {
	component := &ExternalSecrets{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("charts:ExternalSecrets", name, component, opts...)
	if err != nil {
		return nil, err
	}

	releaseArgs := &helm.ReleaseArgs{
		Chart:           pulumi.String("external-secrets"),
		Name:            pulumi.String("external-secrets"),
		Namespace:       pulumi.String("external-secrets"),
		CreateNamespace: pulumi.Bool(true),
		WaitForJobs:     pulumi.BoolPtr(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.external-secrets.io"),
		},
		Values: pulumi.Map{},
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
