package secrets

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DefaultStoreName is the ClusterSecretStore used when none is given.
const DefaultStoreName = "infisical-secret-store"

// DefaultRefreshInterval is the PushSecret refresh interval used when none is given.
const DefaultRefreshInterval = "1h"

// PushMapping maps one key of the source Secret to one or more remote
// secret names — multiple RemoteKeys publish the same value under aliases.
type PushMapping struct {
	SecretKey  string
	RemoteKeys []string
}

type PushSecret struct {
	pulumi.ResourceState

	Resource *apiextensions.CustomResource
}

type PushSecretArgs struct {
	// Namespace of the source Secret and the PushSecret resource.
	Namespace pulumi.StringInput
	// SecretName selects the source Secret to push.
	SecretName pulumi.StringInput
	// StoreName of the ClusterSecretStore (default: infisical-secret-store).
	StoreName string
	// RefreshInterval of the PushSecret (default: 1h).
	RefreshInterval string
	// Mappings is ordered — a stable order keeps pulumi previews diff-free.
	Mappings []PushMapping
}

// NewPushSecret creates an external-secrets PushSecret that publishes keys of
// an existing Secret to the cluster secret store.
func NewPushSecret(ctx *pulumi.Context, name string, args *PushSecretArgs, opts ...pulumi.ResourceOption) (*PushSecret, error) {
	component := &PushSecret{}

	err := ctx.RegisterComponentResource("secrets:PushSecret", name, component, opts...)
	if err != nil {
		return nil, err
	}

	storeName := args.StoreName
	if storeName == "" {
		storeName = DefaultStoreName
	}

	refreshInterval := args.RefreshInterval
	if refreshInterval == "" {
		refreshInterval = DefaultRefreshInterval
	}

	data := pulumi.Array{}
	for _, mapping := range args.Mappings {
		for _, remoteKey := range mapping.RemoteKeys {
			data = append(data, pulumi.Map{
				"match": pulumi.Map{
					"secretKey": pulumi.String(mapping.SecretKey),
					"remoteRef": pulumi.Map{
						"remoteKey": pulumi.String(remoteKey),
					},
				},
			})
		}
	}

	resource, err := apiextensions.NewCustomResource(ctx, name, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("external-secrets.io/v1alpha1"),
		Kind:       pulumi.String("PushSecret"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: args.Namespace,
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"updatePolicy":    pulumi.String("Replace"),
				"deletionPolicy":  pulumi.String("None"),
				"refreshInterval": pulumi.String(refreshInterval),
				"secretStoreRefs": pulumi.Array{
					pulumi.Map{
						"name": pulumi.String(storeName),
						"kind": pulumi.String("ClusterSecretStore"),
					},
				},
				"selector": pulumi.Map{
					"secret": pulumi.Map{
						"name": args.SecretName,
					},
				},
				"data": data,
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Resource = resource

	return component, nil
}
