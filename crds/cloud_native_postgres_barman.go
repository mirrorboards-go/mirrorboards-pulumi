package crds

import (
	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CloudNativePostgresBarman struct {
	pulumi.ResourceState

	ConfigFile *yaml.ConfigFile
}

type NewCloudNativePostgresBarmanArgs struct{}

func NewCloudNativePostgresBarman(ctx *pulumi.Context, name string, args *NewCloudNativePostgresBarmanArgs, opts ...pulumi.ResourceOption) (*CloudNativePostgresBarman, error) {
	component := &CloudNativePostgresBarman{}

	ns := namespace.NewNamespace("crds")

	err := ctx.RegisterComponentResource("platform:crds:CloudNativePostgresBarman", name, component, opts...)

	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &NewCloudNativePostgresBarmanArgs{}
	}

	ConfigFile, err := yaml.NewConfigFile(ctx, ns.Get("cloud-native-postgres-barman"), &yaml.ConfigFileArgs{
		File: "https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v0.10.0/manifest.yaml",
	}, pulumi.Parent(component))

	if err != nil {
		return nil, err
	}

	component.ConfigFile = ConfigFile

	return component, nil
}
