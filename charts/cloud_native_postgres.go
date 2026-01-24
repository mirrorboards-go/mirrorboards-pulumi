package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CloudNativePostgres struct {
	pulumi.ResourceState

	Release *helm.Release
}

type NewCloudNativePostgresArgs struct{}

func NewCloudNativePostgres(ctx *pulumi.Context, name string, args *NewCloudNativePostgresArgs, opts ...pulumi.ResourceOption) (*CloudNativePostgres, error) {
	component := &CloudNativePostgres{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("platform:charts:CloudNativePostgres", name, component, opts...)

	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &NewCloudNativePostgresArgs{}
	}

	Release, err := helm.NewRelease(ctx, ns.Get(), &helm.ReleaseArgs{
		Name:            pulumi.String("cnpg-operator"),
		Namespace:       pulumi.String("cnpg-system"),
		CreateNamespace: pulumi.Bool(true),
		Chart:           pulumi.String("cloudnative-pg"),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://cloudnative-pg.github.io/charts"),
		},
		Values: pulumi.Map{},
	}, pulumi.Parent(component))

	if err != nil {
		return nil, err
	}

	component.Release = Release

	return component, nil
}
