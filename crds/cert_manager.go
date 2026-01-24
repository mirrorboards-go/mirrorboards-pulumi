package crds

import (
	"fmt"

	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// DefaultCertManagerVersion is the default version of cert-manager CRDs
	DefaultCertManagerVersion = "v1.18.2"
)

type CertManagerCRDs struct {
	pulumi.ResourceState

	ConfigFile *yaml.ConfigFile
}

type CertManagerCRDsArgs struct {
	// Version of cert-manager CRDs to install (default: v1.18.2)
	Version string
}

// NewCertManagerCRDs installs cert-manager CRDs from the official GitHub release.
func NewCertManagerCRDs(ctx *pulumi.Context, name string, args *CertManagerCRDsArgs, opts ...pulumi.ResourceOption) (*CertManagerCRDs, error) {
	component := &CertManagerCRDs{}

	ns := namespace.NewNamespace("crds")

	err := ctx.RegisterComponentResource("crds:CertManager", name, component, opts...)
	if err != nil {
		return nil, err
	}

	version := DefaultCertManagerVersion
	if args != nil && args.Version != "" {
		version = args.Version
	}

	url := fmt.Sprintf("https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.crds.yaml", version)

	configFile, err := yaml.NewConfigFile(ctx, ns.Get("cert-manager"), &yaml.ConfigFileArgs{
		File: url,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.ConfigFile = configFile

	return component, nil
}
