package providers

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type DigitalOceanProviderArgs struct {
	// Token overrides the token from config. If nil, uses config secret.
	Token pulumi.StringInput
}

// NewDigitalOceanProvider creates a DigitalOcean provider with IgnoreChanges on token
// to prevent unnecessary diffs when the secret is re-read.
func NewDigitalOceanProvider(ctx *pulumi.Context, name string, args *DigitalOceanProviderArgs, opts ...pulumi.ResourceOption) (*digitalocean.Provider, error) {
	var token pulumi.StringInput

	if args != nil && args.Token != nil {
		token = args.Token
	} else {
		cfg := config.New(ctx, "digitalocean")
		token = cfg.RequireSecret("token")
	}

	opts = append(opts, pulumi.IgnoreChanges([]string{"token"}))

	return digitalocean.NewProvider(ctx, name, &digitalocean.ProviderArgs{
		Token: token,
	}, opts...)
}
