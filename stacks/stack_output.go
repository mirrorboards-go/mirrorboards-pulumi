package stacks

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GetStringStackOutput returns a string output from another stack.
// stackRef format: "organization/project/stack"
func GetStringStackOutput(ctx *pulumi.Context, stackRef string, outputKey string) pulumi.StringOutput {
	stack, err := pulumi.NewStackReference(ctx, stackRef, nil)
	if err != nil {
		panic(err)
	}

	return stack.GetStringOutput(pulumi.String(outputKey))
}
