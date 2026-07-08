package stacks

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EksClusterProviderArgs struct {
	// StackReference to the cluster stack, e.g. "organization/<project>/dev".
	// The stack must export ClusterName, ClusterEndpoint,
	// ClusterCertificateAuthorityData and Region.
	StackReference string
	// RoleArn assumed by the exec credential plugin — this is where the
	// blast radius of the consuming stack group is decided (EKS access
	// entries scope what the role can do inside the cluster).
	RoleArn pulumi.StringInput
	// Profile used by the exec credential plugin to source AWS credentials.
	Profile string
}

// NewEksClusterProvider builds a Kubernetes provider for an EKS cluster from
// the cluster stack's plain exports (endpoint + CA), authenticating through
// `aws eks get-token` with the given role. Unlike the admin kubeconfig
// exported by the cluster stack, no secret material is involved — IAM and
// EKS access entries are the only gate.
func NewEksClusterProvider(ctx *pulumi.Context, name string, args *EksClusterProviderArgs, opts ...pulumi.ResourceOption) (*kubernetes.Provider, error) {
	clusterStack, err := pulumi.NewStackReference(ctx, args.StackReference, nil)
	if err != nil {
		return nil, err
	}

	clusterName := clusterStack.GetStringOutput(pulumi.String("ClusterName"))
	endpoint := clusterStack.GetStringOutput(pulumi.String("ClusterEndpoint"))
	caData := clusterStack.GetStringOutput(pulumi.String("ClusterCertificateAuthorityData"))
	region := clusterStack.GetStringOutput(pulumi.String("Region"))

	profileArgs := ""
	if args.Profile != "" {
		profileArgs = `, "--profile", "` + args.Profile + `"`
	}

	kubeconfig := pulumi.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %[1]s
    certificate-authority-data: %[2]s
  name: %[3]s
contexts:
- context:
    cluster: %[3]s
    user: %[3]s
  name: %[3]s
current-context: %[3]s
users:
- name: %[3]s
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args: ["eks", "get-token", "--cluster-name", "%[3]s", "--region", "%[4]s", "--role-arn", "%[5]s"%[6]s]
`, endpoint, caData, clusterName, region, args.RoleArn, profileArgs)

	return kubernetes.NewProvider(ctx, name, &kubernetes.ProviderArgs{
		Kubeconfig: kubeconfig,
	}, opts...)
}
