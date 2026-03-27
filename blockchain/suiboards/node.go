package suiboards

import (
	"fmt"

	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
)

type Node struct {
	pulumi.ResourceState

	Service *corev1.Service
}

type NodeArgs struct {
	Name          pulumi.StringInput
	Namespace     pulumi.StringInput
	Image         pulumi.StringInput
	CommitteeSize int
	Hostname      pulumi.StringInput
}

func NewNode(ctx *pulumi.Context, name string, args *NodeArgs, opts ...pulumi.ResourceOption) (*Node, error) {
	component := &Node{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("suiboards:network:Node", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &NodeArgs{}
	}

	if args.Image == nil {
		args.Image = pulumi.String("mysten/sui-tools:mainnet")
	}

	if args.CommitteeSize == 0 {
		args.CommitteeSize = 1
	}

	Labels := pulumi.StringMap{
		"app": pulumi.String(ns.Get()),
	}

	entrypointScript := `#!/bin/bash
set -e

CONFIG_DIR="/opt/sui/config"
export RUST_LOG="${RUST_LOG:-off,sui_node=info}"

if [ ! -f "$CONFIG_DIR/genesis.blob" ]; then
  echo "[sui-node] Generating genesis with committee-size=${COMMITTEE_SIZE:-1}..."
  sui genesis \
    --working-dir "$CONFIG_DIR" \
    --committee-size "${COMMITTEE_SIZE:-1}" \
    --force
  echo "[sui-node] Genesis generated at $CONFIG_DIR"
fi

echo "[sui-node] Starting SUI network..."
exec sui start \
  --network.config "$CONFIG_DIR" \
  --with-faucet=0.0.0.0:9123
`

	ConfigMap, err := corev1.NewConfigMap(ctx, ns.Get("entrypoint"), &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ns.Get("entrypoint")),
			Namespace: args.Namespace,
		},
		Data: pulumi.StringMap{
			"entrypoint.sh": pulumi.String(entrypointScript),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	NodePVC, err := corev1.NewPersistentVolumeClaim(ctx, ns.Get("pvc"), &corev1.PersistentVolumeClaimArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ns.Get("pvc")),
			Namespace: args.Namespace,
		},
		Spec: &corev1.PersistentVolumeClaimSpecArgs{
			AccessModes: pulumi.StringArray{
				pulumi.String("ReadWriteOnce"),
			},
			Resources: &corev1.VolumeResourceRequirementsArgs{
				Requests: pulumi.StringMap{
					"storage": pulumi.String("5Gi"),
				},
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	serviceType := "ClusterIP"
	serviceAnnotations := pulumi.StringMap{}

	if args.Hostname != nil {
		serviceType = "LoadBalancer"
		serviceAnnotations = pulumi.StringMap{
			"external-dns.alpha.kubernetes.io/hostname": args.Hostname,
			"external-dns.alpha.kubernetes.io/ttl":      pulumi.String("1800"),
		}
	}

	Service, err := corev1.NewService(ctx, ns.Get("svc"), &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        args.Name,
			Namespace:   args.Namespace,
			Labels:      Labels,
			Annotations: serviceAnnotations,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String(serviceType),
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(9000),
					TargetPort: pulumi.Int(9000),
					Name:       pulumi.String("rpc"),
					Protocol:   pulumi.String("TCP"),
				},
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(9123),
					TargetPort: pulumi.Int(9123),
					Name:       pulumi.String("faucet"),
					Protocol:   pulumi.String("TCP"),
				},
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(9184),
					TargetPort: pulumi.Int(9184),
					Name:       pulumi.String("metrics"),
					Protocol:   pulumi.String("TCP"),
				},
			},
			Selector: Labels,
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Service = Service

	_, err = appsv1.NewDeployment(ctx, ns.Get("deployment"), &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ns.Get("deployment")),
			Namespace: args.Namespace,
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Strategy: &appsv1.DeploymentStrategyArgs{
				Type: pulumi.String("Recreate"),
			},
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: Labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: Labels,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:            pulumi.String("sui-node"),
							Image:           args.Image,
							ImagePullPolicy: pulumi.String("Always"),
							Command: pulumi.StringArray{
								pulumi.String("bash"),
								pulumi.String("/scripts/entrypoint.sh"),
							},
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name:  pulumi.String("COMMITTEE_SIZE"),
									Value: pulumi.String(fmt.Sprintf("%d", args.CommitteeSize)),
								},
								&corev1.EnvVarArgs{
									Name:  pulumi.String("RUST_LOG"),
									Value: pulumi.String("off,sui_node=info"),
								},
							},
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("rpc"),
									ContainerPort: pulumi.Int(9000),
								},
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("faucet"),
									ContainerPort: pulumi.Int(9123),
								},
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("metrics"),
									ContainerPort: pulumi.Int(9184),
								},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("data"),
									MountPath: pulumi.String("/opt/sui"),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("scripts"),
									MountPath: pulumi.String("/scripts"),
									ReadOnly:  pulumi.Bool(true),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("data"),
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
								ClaimName: NodePVC.Metadata.Name().Elem(),
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("scripts"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name:        ConfigMap.Metadata.Name().Elem(),
								DefaultMode: pulumi.Int(0755),
							},
						},
					},
				},
			},
		},
	},
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{NodePVC, ConfigMap}),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}
