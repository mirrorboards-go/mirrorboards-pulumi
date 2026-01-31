package actaboards

import (
	"fmt"
	"strings"

	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
)

type SeedNode struct {
	pulumi.ResourceState

	Service *corev1.Service
}

type SeedNodeArgs struct {
	Name           pulumi.StringInput
	Namespace      pulumi.StringInput
	Hostname       pulumi.StringInput
	GenesisURL     pulumi.StringInput
	SeedNodes      pulumi.StringArrayInput
	CertIssuerName pulumi.StringInput
	Image          pulumi.StringInput
	Plugins        pulumi.StringArrayInput
}

func NewSeedNode(ctx *pulumi.Context, name string, args *SeedNodeArgs, opts ...pulumi.ResourceOption) (*SeedNode, error) {
	component := &SeedNode{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("actaboards:network:SeedNode", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &SeedNodeArgs{}
	}

	if args.Image == nil {
		args.Image = pulumi.String("ghcr.io/actaboards/actaboards-core:latest")
	}

	Labels := pulumi.StringMap{
		"app": pulumi.String(ns.Get()),
	}

	CertSecretName := ns.Get("tls")

	Cert, err := apiextensions.NewCustomResource(ctx, ns.Get("certificate"), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("Certificate"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      args.Name,
			Namespace: args.Namespace,
			Annotations: pulumi.StringMap{
				"pulumi.com/waitFor": pulumi.String("condition=Ready"),
			},
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"secretName": CertSecretName,
				"issuerRef": map[string]interface{}{
					"name": args.CertIssuerName,
					"kind": "ClusterIssuer",
				},
				"dnsNames": []interface{}{
					args.Hostname,
				},
			},
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

	GenesisPVC, err := corev1.NewPersistentVolumeClaim(ctx, ns.Get("genesis-pvc"), &corev1.PersistentVolumeClaimArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ns.Get("genesis-pvc")),
			Namespace: args.Namespace,
		},
		Spec: &corev1.PersistentVolumeClaimSpecArgs{
			AccessModes: pulumi.StringArray{
				pulumi.String("ReadWriteOnce"),
			},
			Resources: &corev1.VolumeResourceRequirementsArgs{
				Requests: pulumi.StringMap{
					"storage": pulumi.String("1Gi"),
				},
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	Service, err := corev1.NewService(ctx, ns.Get("svc"), &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      args.Name,
			Namespace: args.Namespace,
			Labels:    Labels,
			Annotations: pulumi.StringMap{
				"external-dns.alpha.kubernetes.io/hostname": args.Hostname,
				"external-dns.alpha.kubernetes.io/ttl":      pulumi.String("1800"),
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("LoadBalancer"),
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:        pulumi.Int(8090),
					TargetPort:  pulumi.Int(8090),
					Name:        pulumi.String("rpc"),
					Protocol:    pulumi.String("TCP"),
					AppProtocol: pulumi.String("kubernetes.io/wss"),
				},
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

	component.Service = Service

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
					InitContainers: func() corev1.ContainerArray {
						containers := corev1.ContainerArray{}

						containers = append(containers, &corev1.ContainerArgs{
							Name:  pulumi.String("tls-combiner"),
							Image: pulumi.String("busybox:latest"),
							Command: pulumi.StringArray{
								pulumi.String("sh"),
								pulumi.String("-c"),
								pulumi.String("cat /tls-certs/tls.crt /tls-certs/tls.key > /tls/combined.pem"),
							},
							SecurityContext: &corev1.SecurityContextArgs{
								RunAsUser:  pulumi.Int(1000),
								RunAsGroup: pulumi.Int(1000),
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("tls-volume"),
									MountPath: pulumi.String("/tls-certs"),
									ReadOnly:  pulumi.Bool(true),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("tls-writable-volume"),
									MountPath: pulumi.String("/tls"),
								},
							},
						})

						if args.GenesisURL != nil {
							containers = append(containers, &corev1.ContainerArgs{
								Name:  pulumi.String("genesis-downloader"),
								Image: pulumi.String("curlimages/curl:latest"),
								Command: pulumi.StringArray{
									pulumi.String("sh"),
									pulumi.String("-c"),
									pulumi.String("curl -L $(GENESIS_URL) -o /genesis/genesis.json && echo 'Genesis file downloaded successfully'"),
								},
								SecurityContext: &corev1.SecurityContextArgs{
									RunAsUser:  pulumi.Int(1000),
									RunAsGroup: pulumi.Int(1000),
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{
										Name:  pulumi.String("GENESIS_URL"),
										Value: args.GenesisURL,
									},
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("genesis-volume"),
										MountPath: pulumi.String("/genesis"),
									},
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("tls-volume"),
										MountPath: pulumi.String("/tls-certs"),
										ReadOnly:  pulumi.Bool(true),
									},
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("tls-writable-volume"),
										MountPath: pulumi.String("/tls"),
									},
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("data-volume"),
										MountPath: pulumi.String("/data"),
									},
								},
							})
						}

						return containers
					}(),
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
									pulumi.String("--rpc-tls-endpoint=0.0.0.0:8090"),
									pulumi.String("--server-pem=/tls/combined.pem"),
								}

								if args.GenesisURL != nil {
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

								if args.Plugins != nil {
									pluginsArg := args.Plugins.ToStringArrayOutput().ApplyT(func(plugins []string) string {
										if len(plugins) > 0 {
											return fmt.Sprintf("--plugins=%s", strings.Join(plugins, " "))
										}
										return ""
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, pluginsArg)
								}

								return baseArgs
							}(),
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("rpc"),
									ContainerPort: pulumi.Int(8090),
								},
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
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("tls-writable-volume"),
										MountPath: pulumi.String("/tls"),
									},
								}

								if args.GenesisURL != nil {
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
							&corev1.VolumeArgs{
								Name: pulumi.String("tls-volume"),
								Secret: &corev1.SecretVolumeSourceArgs{
									SecretName: pulumi.String(CertSecretName),
								},
							},
							&corev1.VolumeArgs{
								Name:     pulumi.String("tls-writable-volume"),
								EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
							},
						}

						if args.GenesisURL != nil {
							volumes = append(volumes, &corev1.VolumeArgs{
								Name: pulumi.String("genesis-volume"),
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: GenesisPVC.Metadata.Name().Elem().ToStringOutput(),
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
		pulumi.DependsOn([]pulumi.Resource{NodePVC, GenesisPVC, Cert}),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}
