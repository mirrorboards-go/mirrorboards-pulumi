package stacks

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/providers"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type DigitalOceanCluster struct {
	pulumi.ResourceState

	Provider *kubernetes.Provider
}

type DigitalOceanClusterLookupArgs struct {
	// Name of the cluster to lookup
	Name pulumi.StringInput
}

// NewDigitalOceanClusterLookup creates a Kubernetes provider from an existing DigitalOcean cluster.
func NewDigitalOceanClusterLookup(ctx *pulumi.Context, name string, args *DigitalOceanClusterLookupArgs, opts ...pulumi.ResourceOption) (*DigitalOceanCluster, error) {
	component := &DigitalOceanCluster{}

	err := ctx.RegisterComponentResource("stacks:DigitalOceanCluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	doProvider, err := providers.NewDigitalOceanProvider(ctx, name+"-do-provider", nil, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	kubeconfig := args.Name.ToStringOutput().ApplyT(func(clusterName string) (*string, error) {
		cluster, err := digitalocean.LookupKubernetesCluster(ctx, &digitalocean.LookupKubernetesClusterArgs{
			Name: clusterName,
		}, pulumi.Provider(doProvider))
		if err != nil {
			return nil, err
		}

		config := cluster.KubeConfigs[0].RawConfig
		return &config, nil
	}).(pulumi.StringPtrOutput)

	k8sProvider, err := kubernetes.NewProvider(ctx, name+"-k8s-provider", &kubernetes.ProviderArgs{
		Kubeconfig: kubeconfig,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Provider = k8sProvider

	return component, nil
}

type DigitalOceanClusterFromStackArgs struct {
	// StackReference in format "organization/project/stack"
	StackReference string
	// OutputKey is the name of the output containing cluster name (default: "ClusterName")
	OutputKey string
}

// NewDigitalOceanClusterFromStack creates a Kubernetes provider from a cluster defined in another Pulumi stack.
func NewDigitalOceanClusterFromStack(ctx *pulumi.Context, name string, args *DigitalOceanClusterFromStackArgs, opts ...pulumi.ResourceOption) (*DigitalOceanCluster, error) {
	stackRef, err := pulumi.NewStackReference(ctx, args.StackReference, nil)
	if err != nil {
		return nil, err
	}

	outputKey := args.OutputKey
	if outputKey == "" {
		outputKey = "ClusterName"
	}

	clusterName := stackRef.GetStringOutput(pulumi.String(outputKey))

	return NewDigitalOceanClusterLookup(ctx, name, &DigitalOceanClusterLookupArgs{
		Name: clusterName,
	}, opts...)
}
