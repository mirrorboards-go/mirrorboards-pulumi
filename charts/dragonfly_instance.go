package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
)

type DragonflyInstance struct {
	pulumi.ResourceState

	ComponentResource *apiextensions.CustomResource
}

type NewDragonflyInstanceArgs struct {
	Name      pulumi.StringInput
	Namespace pulumi.StringInput
}

func NewDragonflyInstance(ctx *pulumi.Context, name string, args *NewDragonflyInstanceArgs, opts ...pulumi.ResourceOption) (*DragonflyInstance, error) {
	component := &DragonflyInstance{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("blockchain:network:DragonflyInstance", name, component, opts...)

	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &NewDragonflyInstanceArgs{}
	}

	Dragonfly, err := apiextensions.NewCustomResource(ctx, ns.Get("dragonfly"), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("dragonflydb.io/v1alpha1"),
		Kind:       pulumi.String("Dragonfly"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      args.Name,
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name":       pulumi.String("dragonfly"),
				"app.kubernetes.io/instance":   pulumi.String("dragonfly-sample"),
				"app.kubernetes.io/part-of":    pulumi.String("dragonfly-operator"),
				"app.kubernetes.io/managed-by": pulumi.String("kustomize"),
				"app.kubernetes.io/created-by": pulumi.String("dragonfly-operator"),
			},
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": 1,
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    "500m",
						"memory": "500Mi",
					},
					"limits": map[string]interface{}{
						"cpu":    "600m",
						"memory": "750Mi",
					},
				},
			},
		},
	}, pulumi.Parent(component))

	if err != nil {
		return nil, err
	}

	component.ComponentResource = Dragonfly

	return component, nil
}
