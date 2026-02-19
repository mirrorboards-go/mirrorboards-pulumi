package actaboards

import (
	"fmt"
	"strings"

	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
)

type WitnessNodeWithSharedPVC struct {
	pulumi.ResourceState
}

type WitnessNodeWithSharedPVCArgs struct {
	Name           pulumi.StringInput
	Namespace      pulumi.StringInput
	GenesisPVCName pulumi.StringInput
	SeedNodes      pulumi.StringArrayInput
	Witness        *WitnessArgs
	Image          pulumi.StringInput
}

func NewWitnessNodeWithSharedPVC(ctx *pulumi.Context, name string, args *WitnessNodeWithSharedPVCArgs, opts ...pulumi.ResourceOption) (*WitnessNodeWithSharedPVC, error) {
	component := &WitnessNodeWithSharedPVC{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("actaboards:network:WitnessNodeWithSharedPVC", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &WitnessNodeWithSharedPVCArgs{}
	}

	if args.Image == nil {
		args.Image = pulumi.String("ghcr.io/actaboards/actaboards-core:latest")
	}

	Labels := pulumi.StringMap{
		"app": pulumi.String(ns.Get()),
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

	_, err = corev1.NewService(ctx, ns.Get("svc"), &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      args.Name,
			Namespace: args.Namespace,
			Labels:    Labels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("ClusterIP"),
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:        pulumi.Int(2771),
					TargetPort:  pulumi.Int(2771),
					Name:        pulumi.String("p2p"),
					Protocol:    pulumi.String("TCP"),
					AppProtocol: pulumi.String("kubernetes.io/wss"),
				},
			},
			Selector: Labels,
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	_, err = appsv1.NewDeployment(ctx, ns.Get("deployment"), &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ns.Get("deployment")),
			Namespace: args.Namespace,
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: Labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: Labels,
				},
				Spec: &corev1.PodSpecArgs{
					SecurityContext: &corev1.PodSecurityContextArgs{
						FsGroup: pulumi.Int(1000),
					},
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:            pulumi.String(ns.Get("rpc")),
							Image:           args.Image,
							ImagePullPolicy: pulumi.String("Always"),
							Command: pulumi.StringArray{
								pulumi.String("/usr/local/bin/witness_node"),
							},
							Args: func() pulumi.StringArray {
								baseArgs := pulumi.StringArray{
									pulumi.String("--data-dir=/data"),
									pulumi.String("--p2p-endpoint=0.0.0.0:2771"),
								}

								if args.Witness != nil && args.Witness.ID != nil {
									baseArgs = append(baseArgs, pulumi.Sprintf("--witness-id=\"%s\"", args.Witness.ID))
								}

								if args.Witness != nil && args.Witness.PublicKey != nil && args.Witness.PrivateKey != nil {
									baseArgs = append(baseArgs, pulumi.Sprintf("--private-key=[\"%s\", \"%s\"]", args.Witness.PublicKey, args.Witness.PrivateKey))
								}

								if args.Witness != nil && args.Witness.EnableStaleProduction {
									baseArgs = append(baseArgs, pulumi.String("--enable-stale-production"))
								}

								if args.GenesisPVCName != nil {
									baseArgs = append(baseArgs, pulumi.String("--genesis-json=/genesis/genesis.json"))
								}

								if args.SeedNodes != nil {
									seedNodeTLSsArg := args.SeedNodes.ToStringArrayOutput().ApplyT(func(nodes []string) string {
										nodeList := "[]"
										if len(nodes) > 0 {
											nodeList = fmt.Sprintf(`["%s"]`, strings.Join(nodes, `", "`))
										}
										return fmt.Sprintf("--seed-nodes=%s", nodeList)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, seedNodeTLSsArg)
								}
								return baseArgs
							}(),
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("p2p"),
									ContainerPort: pulumi.Int(2771),
								},
							},
							VolumeMounts: func() corev1.VolumeMountArray {
								volumeMounts := corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("data-volume"),
										MountPath: pulumi.String("/data"),
									},
								}

								if args.GenesisPVCName != nil {
									volumeMounts = append(volumeMounts, &corev1.VolumeMountArgs{
										Name:      pulumi.String("genesis-volume"),
										MountPath: pulumi.String("/genesis"),
										ReadOnly:  pulumi.Bool(true),
									})
								}

								return volumeMounts
							}(),
						},
					},
					Volumes: func() corev1.VolumeArray {
						volumes := corev1.VolumeArray{
							&corev1.VolumeArgs{
								Name: pulumi.String("data-volume"),
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: NodePVC.Metadata.Name().Elem(),
								},
							},
						}

						if args.GenesisPVCName != nil {
							volumes = append(volumes, &corev1.VolumeArgs{
								Name: pulumi.String("genesis-volume"),
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: args.GenesisPVCName,
								},
							})
						}

						return volumes
					}(),
				},
			},
		},
	},
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{NodePVC}),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}
