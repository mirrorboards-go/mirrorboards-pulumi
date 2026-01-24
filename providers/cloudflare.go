package providers

import (
	"github.com/pulumi/pulumi-cloudflare/sdk/v6/go/cloudflare"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type CloudflareProviderArgs struct {
	// ApiToken overrides the token from config. If nil, uses config secret.
	ApiToken pulumi.StringInput
}

// NewCloudflareProvider creates a Cloudflare provider with IgnoreChanges on apiToken
// to prevent unnecessary diffs when the secret is re-read.
func NewCloudflareProvider(ctx *pulumi.Context, name string, args *CloudflareProviderArgs, opts ...pulumi.ResourceOption) (*cloudflare.Provider, error) {
	var token pulumi.StringInput

	if args != nil && args.ApiToken != nil {
		token = args.ApiToken
	} else {
		cfg := config.New(ctx, "cloudflare")
		token = cfg.RequireSecret("token")
	}

	opts = append(opts, pulumi.IgnoreChanges([]string{"apiToken"}))

	return cloudflare.NewProvider(ctx, name, &cloudflare.ProviderArgs{
		ApiToken: token,
	}, opts...)
}
