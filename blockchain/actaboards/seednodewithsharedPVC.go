package actaboards

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mirrorboards-go/mirrorboards-pulumi/namespace"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
)

type SeedNodeWithSharedPVC struct {
	pulumi.ResourceState

	Service *corev1.Service
}

type SeedNodeWithSharedPVCArgs struct {
	Name             pulumi.StringInput
	Namespace        pulumi.StringInput
	NodeSelector     pulumi.StringMapInput
	Tolerations      corev1.TolerationArrayInput
	Hostname         pulumi.StringInput
	GenesisPVCName   pulumi.StringInput
	GenesisPublicURL pulumi.StringInput
	SeedNodes        pulumi.StringArrayInput
	CertIssuerName   pulumi.StringInput
	Image            pulumi.StringInput
	Plugins          pulumi.StringArrayInput
	// AllowedAPIs overrides the anonymous RPC API permissions when provided.
	AllowedAPIs pulumi.StringArrayInput
	ExposeRPC   *bool
}

func NewSeedNodeWithSharedPVC(ctx *pulumi.Context, name string, args *SeedNodeWithSharedPVCArgs, opts ...pulumi.ResourceOption) (*SeedNodeWithSharedPVC, error) {
	component := &SeedNodeWithSharedPVC{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("actaboards:network:SeedNodeWithSharedPVC", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &SeedNodeWithSharedPVCArgs{}
	}

	if args.Image == nil {
		args.Image = pulumi.String("ghcr.io/actaboards/actaboards-core:latest")
	}

	exposeRPC := true
	if args.ExposeRPC != nil {
		exposeRPC = *args.ExposeRPC
	}

	if args.GenesisPVCName != nil && args.GenesisPublicURL != nil {
		return nil, fmt.Errorf("seed node %s cannot use both GenesisPVCName and GenesisPublicURL", name)
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
			Ports: func() *corev1.ServicePortArray {
				ports := corev1.ServicePortArray{}

				if exposeRPC {
					ports = append(ports, &corev1.ServicePortArgs{
						Port:        pulumi.Int(8090),
						TargetPort:  pulumi.Int(8090),
						Name:        pulumi.String("rpc"),
						Protocol:    pulumi.String("TCP"),
						AppProtocol: pulumi.String("kubernetes.io/wss"),
					})
				}

				ports = append(ports, &corev1.ServicePortArgs{
					Port:        pulumi.Int(2771),
					TargetPort:  pulumi.Int(2771),
					Name:        pulumi.String("p2p"),
					Protocol:    pulumi.String("TCP"),
					AppProtocol: pulumi.String("kubernetes.io/wss"),
				})

				return &ports
			}(),
			Selector: Labels,
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}

	component.Service = Service

	var apiAccessConfig *corev1.ConfigMap
	podAnnotations := pulumi.StringMap{}
	if args.AllowedAPIs != nil {
		apiAccessJSON := args.AllowedAPIs.ToStringArrayOutput().ApplyT(func(apis []string) (string, error) {
			data, err := json.Marshal(map[string]interface{}{
				"permission_map": [][]interface{}{{
					"*", map[string]interface{}{
						"password_hash_b64": "*",
						"password_salt_b64": "*",
						"allowed_apis":      apis,
					},
				}},
			})
			return string(data), err
		}).(pulumi.StringOutput)
		apiAccessConfig, err = corev1.NewConfigMap(ctx, ns.Get("api-access"), &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(ns.Get("api-access")),
				Namespace: args.Namespace,
			},
			Data: pulumi.StringMap{"api-access.json": apiAccessJSON},
		}, pulumi.Parent(component))
		if err != nil {
			return nil, err
		}
		// Permissions are read at startup, so changes must restart the node.
		podAnnotations["checksum/api-access"] = apiAccessJSON.ApplyT(func(value string) string {
			return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
		}).(pulumi.StringOutput)
	}

	_, err = appsv1.NewDeployment(ctx, ns.Get("deployment"), &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(ns.Get("deployment")),
			Namespace: args.Namespace,
		},
		Spec: &appsv1.DeploymentSpecArgs{
			// A chain database on one PVC must never have two concurrent writers.
			Strategy: &appsv1.DeploymentStrategyArgs{
				Type: pulumi.String("RollingUpdate"),
				RollingUpdate: &appsv1.RollingUpdateDeploymentArgs{
					MaxSurge:       pulumi.Int(0),
					MaxUnavailable: pulumi.Int(1),
				},
			},
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: Labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:      Labels,
					Annotations: podAnnotations,
				},
				Spec: &corev1.PodSpecArgs{
					NodeSelector: args.NodeSelector,
					Tolerations:  args.Tolerations,
					SecurityContext: &corev1.PodSecurityContextArgs{
						FsGroup: pulumi.Int(1000),
					},
					InitContainers: func() corev1.ContainerArray {
						initContainers := corev1.ContainerArray{
							&corev1.ContainerArgs{
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
							},
						}

						if args.GenesisPublicURL != nil {
							initContainers = append(initContainers, &corev1.ContainerArgs{
								Name:  pulumi.String("genesis-bootstrap"),
								Image: pulumi.String("curlimages/curl:latest"),
								Command: pulumi.StringArray{
									pulumi.String("sh"),
									pulumi.String("-ec"),
									pulumi.Sprintf("curl --fail --silent --show-error --location %q -o /genesis/genesis.json && test -s /genesis/genesis.json", args.GenesisPublicURL),
								},
								SecurityContext: &corev1.SecurityContextArgs{
									RunAsUser:  pulumi.Int(1000),
									RunAsGroup: pulumi.Int(1000),
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("genesis-volume"),
										MountPath: pulumi.String("/genesis"),
									},
								},
							})
						}

						return initContainers
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
									pulumi.String("--server-pem=/tls/combined.pem"),
								}

								if apiAccessConfig != nil {
									baseArgs = append(baseArgs, pulumi.String("--api-access=/api-access/api-access.json"))
								}

								if exposeRPC {
									baseArgs = append(baseArgs, pulumi.String("--rpc-tls-endpoint=0.0.0.0:8090"))
								}

								if args.GenesisPVCName != nil || args.GenesisPublicURL != nil {
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
							Ports: func() corev1.ContainerPortArray {
								ports := corev1.ContainerPortArray{}

								if exposeRPC {
									ports = append(ports, &corev1.ContainerPortArgs{
										Name:          pulumi.String("rpc"),
										ContainerPort: pulumi.Int(8090),
									})
								}

								ports = append(ports, &corev1.ContainerPortArgs{
									Name:          pulumi.String("p2p"),
									ContainerPort: pulumi.Int(2771),
								})

								return ports
							}(),
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

								if args.GenesisPVCName != nil || args.GenesisPublicURL != nil {
									volumeMounts = append(volumeMounts, &corev1.VolumeMountArgs{
										Name:      pulumi.String("genesis-volume"),
										MountPath: pulumi.String("/genesis"),
										ReadOnly:  pulumi.Bool(true),
									})
								}

								if apiAccessConfig != nil {
									volumeMounts = append(volumeMounts, &corev1.VolumeMountArgs{
										Name:      pulumi.String("api-access"),
										MountPath: pulumi.String("/api-access"),
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

						if args.GenesisPVCName != nil {
							volumes = append(volumes, &corev1.VolumeArgs{
								Name: pulumi.String("genesis-volume"),
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
									ClaimName: args.GenesisPVCName,
								},
							})
						} else if args.GenesisPublicURL != nil {
							volumes = append(volumes, &corev1.VolumeArgs{
								Name:     pulumi.String("genesis-volume"),
								EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
							})
						}

						if apiAccessConfig != nil {
							volumes = append(volumes, &corev1.VolumeArgs{
								Name: pulumi.String("api-access"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name: apiAccessConfig.Metadata.Name().Elem(),
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
		pulumi.DependsOn([]pulumi.Resource{NodePVC, Cert}),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}
