package stacks

import (
	"encoding/base64"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ClusterProviderArgs struct {
	// StackReference to the cluster stack, e.g. "organization/<project>/dev".
	StackReference string
	// OutputKey holding the base64-encoded kubeconfig (default: "Kubeconfig").
	OutputKey string
}

// NewClusterProvider builds a Kubernetes provider from the base64-encoded
// kubeconfig exported by a cluster stack.
func NewClusterProvider(ctx *pulumi.Context, name string, args *ClusterProviderArgs, opts ...pulumi.ResourceOption) (*kubernetes.Provider, error) {
	outputKey := args.OutputKey
	if outputKey == "" {
		outputKey = "Kubeconfig"
	}

	clusterStack, err := pulumi.NewStackReference(ctx, args.StackReference, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clusterStack.GetStringOutput(pulumi.String(outputKey)).ApplyT(func(encoded string) string {
		decoded, _ := base64.StdEncoding.DecodeString(encoded)
		return string(decoded)
	}).(pulumi.StringOutput)

	return kubernetes.NewProvider(ctx, name, &kubernetes.ProviderArgs{
		Kubeconfig: kubeconfig,
	}, opts...)
}
