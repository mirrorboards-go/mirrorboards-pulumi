package scheduling

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ChainpoolNetworkNodeSelector(network string) pulumi.StringMap {
	return pulumi.StringMap{
		"chainpool.network": pulumi.String(network),
	}
}

func ChainpoolNetworkTolerations(network string) corev1.TolerationArray {
	return corev1.TolerationArray{
		&corev1.TolerationArgs{
			Key:      pulumi.String("chainpool.network"),
			Operator: pulumi.String("Equal"),
			Value:    pulumi.String(network),
			Effect:   pulumi.String("NoSchedule"),
		},
	}
}
