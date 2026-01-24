package charts

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type DragonflyOperator struct {
	pulumi.ResourceState

	ConfigFile *yaml.ConfigFile
}

type NewDragonflyOperatorArgs struct{}

func NewDragonflyOperator(ctx *pulumi.Context, name string, args *NewDragonflyOperatorArgs, opts ...pulumi.ResourceOption) (*DragonflyOperator, error) {
	component := &DragonflyOperator{}

	ns := namespace.NewNamespace("crds")

	err := ctx.RegisterComponentResource("platform:crds:DragonflyOperator", name, component, opts...)

	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &NewDragonflyOperatorArgs{}
	}

	ConfigFile, err := yaml.NewConfigFile(ctx, ns.Get("dragonfly", "operator"), &yaml.ConfigFileArgs{
		File: "https://raw.githubusercontent.com/dragonflydb/dragonfly-operator/main/manifests/dragonfly-operator.yaml",
	}, pulumi.Parent(component))

	if err != nil {
		return nil, err
	}

	component.ConfigFile = ConfigFile

	return component, nil
}
