package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ElasticCloudKubernetes struct {
	pulumi.ResourceState

	Release *helm.Release
}

type NewElasticCloudKubernetesArgs struct{}

func NewElasticCloudKubernetes(ctx *pulumi.Context, name string, args *NewElasticCloudKubernetesArgs, opts ...pulumi.ResourceOption) (*ElasticCloudKubernetes, error) {
	component := &ElasticCloudKubernetes{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("platform:charts:ElasticCloudKubernetes", name, component, opts...)

	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &NewElasticCloudKubernetesArgs{}
	}

	Release, err := helm.NewRelease(ctx, ns.Get(), &helm.ReleaseArgs{
		Name:            pulumi.String("eck-operator"),
		Namespace:       pulumi.String("elastic-system"),
		CreateNamespace: pulumi.Bool(true),
		Chart:           pulumi.String("eck-operator"),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://helm.elastic.co"),
		},
		Values: pulumi.Map{},
	}, pulumi.Parent(component))

	if err != nil {
		return nil, err
	}

	component.Release = Release

	return component, nil
}
