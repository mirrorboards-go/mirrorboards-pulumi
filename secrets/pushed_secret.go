package secrets

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// PushedEntry is one key of the created Secret together with the remote
// secret names it is published under.
type PushedEntry struct {
	Key        string
	Value      pulumi.StringInput
	RemoteKeys []string
}

type PushedSecret struct {
	pulumi.ResourceState

	Secret *corev1.Secret
	// PushSecret is the first store's PushSecret, kept for callers that
	// publish to a single store; PushSecrets holds all of them.
	PushSecret  *PushSecret
	PushSecrets []*PushSecret
}

type PushedSecretArgs struct {
	// Namespace for the Secret and its PushSecret.
	Namespace pulumi.StringInput
	// SecretName of the created Secret (default: the resource name).
	SecretName pulumi.StringInput
	// StoreName of the ClusterSecretStore (default: infisical-secret-store).
	StoreName string
	// StoreNames publishes the SAME values to several stores — one Secret in
	// the cluster, one PushSecret per destination. Used to keep a second
	// Infisical environment (dev) in step with prod without a second owner
	// of the value: the stack that owns a secret owns it everywhere.
	//
	// Takes precedence over StoreName when both are set.
	StoreNames []string
	// RefreshInterval of the PushSecret (default: 1h).
	RefreshInterval string
	// Entries is ordered — a stable order keeps pulumi previews diff-free.
	Entries []PushedEntry
}

// NewPushedSecret creates a Secret from the given entries and a PushSecret
// that publishes them to the cluster secret store.
func NewPushedSecret(ctx *pulumi.Context, name string, args *PushedSecretArgs, opts ...pulumi.ResourceOption) (*PushedSecret, error) {
	component := &PushedSecret{}

	err := ctx.RegisterComponentResource("secrets:PushedSecret", name, component, opts...)
	if err != nil {
		return nil, err
	}

	secretName := args.SecretName
	if secretName == nil {
		secretName = pulumi.String(name)
	}

	stringData := pulumi.StringMap{}
	mappings := make([]PushMapping, 0, len(args.Entries))
	for _, entry := range args.Entries {
		stringData[entry.Key] = entry.Value
		mappings = append(mappings, PushMapping{
			SecretKey:  entry.Key,
			RemoteKeys: entry.RemoteKeys,
		})
	}

	secret, err := corev1.NewSecret(ctx, name, &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      secretName,
			Namespace: args.Namespace,
		},
		StringData: stringData,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	stores := args.StoreNames
	if len(stores) == 0 {
		stores = []string{args.StoreName}
	}

	for index, store := range stores {
		// The first destination keeps the historical resource name, so adding
		// a second store does not replace the PushSecret that already exists.
		pushName := name + "-push"
		if index > 0 {
			pushName = name + "-push-" + store
		}

		pushSecret, err := NewPushSecret(ctx, pushName, &PushSecretArgs{
			Namespace:       args.Namespace,
			SecretName:      secretName,
			StoreName:       store,
			RefreshInterval: args.RefreshInterval,
			Mappings:        mappings,
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{secret}))
		if err != nil {
			return nil, err
		}

		component.PushSecrets = append(component.PushSecrets, pushSecret)
	}

	component.Secret = secret
	component.PushSecret = component.PushSecrets[0]

	return component, nil
}
