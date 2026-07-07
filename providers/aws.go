package providers

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type AwsProviderArgs struct {
	// Region overrides the region from config. If nil, uses aws:region config.
	Region pulumi.StringInput
	// Profile overrides the profile from config. If nil, uses aws:profile config.
	Profile pulumi.StringInput
	// AssumeRoleArn assumes the given role. If nil, uses the ambient identity.
	AssumeRoleArn pulumi.StringInput
	// SessionName names the assume-role session. Defaults to "pulumi-deploy".
	SessionName string
}

// NewAwsProvider creates an AWS provider, optionally assuming a role in another
// account. Region and profile fall back to the aws:* stack config.
func NewAwsProvider(ctx *pulumi.Context, name string, args *AwsProviderArgs, opts ...pulumi.ResourceOption) (*aws.Provider, error) {
	if args == nil {
		args = &AwsProviderArgs{}
	}

	cfg := config.New(ctx, "aws")

	region := args.Region
	if region == nil {
		region = pulumi.String(cfg.Require("region"))
	}

	profile := args.Profile
	if profile == nil {
		if p := cfg.Get("profile"); p != "" {
			profile = pulumi.String(p)
		}
	}

	providerArgs := &aws.ProviderArgs{
		Region: region,
	}
	if profile != nil {
		providerArgs.Profile = profile
	}

	if args.AssumeRoleArn != nil {
		sessionName := args.SessionName
		if sessionName == "" {
			sessionName = "pulumi-deploy"
		}
		providerArgs.AssumeRole = &aws.ProviderAssumeRoleArgs{
			RoleArn:     args.AssumeRoleArn,
			SessionName: pulumi.String(sessionName),
		}
	}

	return aws.NewProvider(ctx, name, providerArgs, opts...)
}
