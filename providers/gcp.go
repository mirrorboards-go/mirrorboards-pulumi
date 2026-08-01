package providers

import (
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type GcpProviderArgs struct {
	// Project overrides the project from config. If nil, uses gcp:project config.
	// Stacks that only touch organization-level resources may leave both unset.
	Project pulumi.StringInput
	// Region overrides the region from config. If nil, uses gcp:region config.
	Region pulumi.StringInput
	// Zone overrides the zone from config. If nil, uses gcp:zone config when set.
	Zone pulumi.StringInput
	// ImpersonateServiceAccount acts as the given service account. If nil, uses
	// the ambient identity — either the operator's ADC or the workload identity
	// of the pod running the program.
	ImpersonateServiceAccount pulumi.StringInput
	// BillingProject names the project quota and billing are charged to, and
	// turns on the `X-Goog-User-Project` header that carries it.
	//
	// Organization-level APIs need this. A call that creates an org policy
	// touches no project, so Google has nothing to bill it to and refuses with
	// a 403 that talks about ADC rather than about the missing project —
	// setting a quota project on the credentials alone does not help, because
	// the provider has to send the header itself. If nil, uses the
	// gcp:billingProject config.
	BillingProject pulumi.StringInput
}

// NewGcpProvider creates a GCP provider, optionally impersonating a service
// account. Project, region and zone fall back to the gcp:* stack config.
//
// Impersonation rather than service account keys is deliberate: an org policy
// forbids key creation, so a long-lived credential is not something a stack can
// obtain even by mistake. What a program may do follows from who runs it.
func NewGcpProvider(ctx *pulumi.Context, name string, args *GcpProviderArgs, opts ...pulumi.ResourceOption) (*gcp.Provider, error) {
	if args == nil {
		args = &GcpProviderArgs{}
	}

	cfg := config.New(ctx, "gcp")

	providerArgs := &gcp.ProviderArgs{}

	// Organization-level stacks legitimately have no project: folders and org
	// policies do not live in one. Requiring it here would force every such
	// stack to name an unrelated project just to satisfy the provider.
	project := args.Project
	if project == nil {
		if p := cfg.Get("project"); p != "" {
			project = pulumi.String(p)
		}
	}
	if project != nil {
		providerArgs.Project = project
	}

	region := args.Region
	if region == nil {
		if r := cfg.Get("region"); r != "" {
			region = pulumi.String(r)
		}
	}
	if region != nil {
		providerArgs.Region = region
	}

	zone := args.Zone
	if zone == nil {
		if z := cfg.Get("zone"); z != "" {
			zone = pulumi.String(z)
		}
	}
	if zone != nil {
		providerArgs.Zone = zone
	}

	if args.ImpersonateServiceAccount != nil {
		providerArgs.ImpersonateServiceAccount = args.ImpersonateServiceAccount
	}

	billingProject := args.BillingProject
	if billingProject == nil {
		if b := cfg.Get("billingProject"); b != "" {
			billingProject = pulumi.String(b)
		}
	}
	if billingProject != nil {
		providerArgs.BillingProject = billingProject
		// Bez tego `billingProject` jest ustawiony, ale nie wysyłany — obie
		// wartości muszą iść razem, inaczej nic się nie zmienia.
		providerArgs.UserProjectOverride = pulumi.Bool(true)
	}

	return gcp.NewProvider(ctx, name, providerArgs, opts...)
}
