// Package azure builds the Azure provider, and is its own package for a reason
// that is not organisational.
//
// Go compiles a package whole. With every cloud's provider in one package, a
// program importing it to get Azure also compiles the AWS, GCP, Cloudflare and
// DigitalOcean SDKs — measured at 9.2 GB of peak memory for a single module,
// against the 7 GB a private repository's CI runner has. Splitting per cloud
// means a platform pays for the clouds it uses.
package azure

import (
	azurenative "github.com/pulumi/pulumi-azure-native-sdk/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type ProviderArgs struct {
	// SubscriptionId overrides the subscription from config. If nil, uses the
	// azure-native:subscriptionId config.
	SubscriptionId pulumi.StringInput
	// TenantId overrides the tenant from config. If nil, uses the
	// azure-native:tenantId config, falling back to whatever the ambient
	// credential resolves to.
	TenantId pulumi.StringInput
	// Location is the default region for resources that do not name one
	// themselves. If nil, uses the azure-native:location config.
	Location pulumi.StringInput
}

// NewProvider creates an Azure provider that authenticates with the
// ambient identity: a federated OIDC token in CI, or the operator's `az login`
// session locally.
//
// There is deliberately no way to pass a client secret. The same reasoning as
// on GCP applies, only the mechanism differs: a long-lived credential is not
// something a stack should be able to obtain, even by mistake. What a program
// may do follows from who runs it, and both paths leave a trace in a log we do
// not write ourselves.
//
// Note that OIDC is not switched on here. The Azure SDK picks it up from the
// environment that GitHub Actions injects (ARM_USE_OIDC and friends), so
// forcing it would break local runs, where the same code must work against an
// interactive session.
func NewProvider(ctx *pulumi.Context, name string, args *ProviderArgs, opts ...pulumi.ResourceOption) (*azurenative.Provider, error) {
	if args == nil {
		args = &ProviderArgs{}
	}

	cfg := config.New(ctx, "azure-native")

	providerArgs := &azurenative.ProviderArgs{}

	subscription := args.SubscriptionId
	if subscription == nil {
		if s := cfg.Get("subscriptionId"); s != "" {
			subscription = pulumi.String(s)
		}
	}
	if subscription != nil {
		providerArgs.SubscriptionId = subscription.ToStringOutput().ApplyT(
			func(v string) *string { return &v },
		).(pulumi.StringPtrOutput)
	}

	tenant := args.TenantId
	if tenant == nil {
		if t := cfg.Get("tenantId"); t != "" {
			tenant = pulumi.String(t)
		}
	}
	if tenant != nil {
		providerArgs.TenantId = tenant.ToStringOutput().ApplyT(
			func(v string) *string { return &v },
		).(pulumi.StringPtrOutput)
	}

	// A default location spares every resource from repeating it, and keeps the
	// region in one place — which matters here, because the region is a product
	// decision (data residency), not a deployment detail.
	location := args.Location
	if location == nil {
		if l := cfg.Get("location"); l != "" {
			location = pulumi.String(l)
		}
	}
	if location != nil {
		providerArgs.Location = location.ToStringOutput().ApplyT(
			func(v string) *string { return &v },
		).(pulumi.StringPtrOutput)
	}

	return azurenative.NewProvider(ctx, name, providerArgs, opts...)
}
