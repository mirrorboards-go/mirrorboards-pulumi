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

type Indexer struct {
	pulumi.ResourceState

	Deployment *appsv1.Deployment
}

type IndexerArgs struct {
	Name                   pulumi.StringInput
	Namespace              pulumi.StringInput
	GenesisURL             pulumi.StringInput
	SeedNodes              pulumi.StringArrayInput
	Image                  pulumi.StringInput
	Plugins                pulumi.StringArrayInput
	PostgresSecretName     pulumi.StringInput
	PostgresSecretKey      pulumi.StringInput
	PostgresStartBlock     pulumi.IntInput
	// Elasticsearch options
	ElasticsearchURL         pulumi.StringInput
	ElasticsearchSecretName  pulumi.StringInput
	ElasticsearchIndexPrefix pulumi.StringInput
	ElasticsearchStartBlock  pulumi.IntInput
	// Postgres indexer options (unified plugin replacing elasticsearch + es_objects + postgres_content)
	PostgresIndexerSecretName     pulumi.StringInput
	PostgresIndexerSecretKey      pulumi.StringInput
	PostgresIndexerMode           pulumi.IntInput
	PostgresIndexerStartBlock     pulumi.IntInput
	PostgresIndexerVisitor        pulumi.BoolInput
	PostgresIndexerOperationString pulumi.BoolInput
}

func NewIndexer(ctx *pulumi.Context, name string, args *IndexerArgs, opts ...pulumi.ResourceOption) (*Indexer, error) {
	component := &Indexer{}

	ns := namespace.NewNamespace(name)

	err := ctx.RegisterComponentResource("actaboards:network:Indexer", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if args == nil {
		args = &IndexerArgs{}
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
					"storage": pulumi.String("10Gi"),
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

	Deployment, err := appsv1.NewDeployment(ctx, ns.Get("deployment"), &appsv1.DeploymentArgs{
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
							Name:  pulumi.String(ns.Get("indexer")),
							Image: args.Image,
							Command: pulumi.StringArray{
								pulumi.String("/bin/sh"),
								pulumi.String("-c"),
								pulumi.String("exec /usr/local/bin/witness_node \"$@\" ${POSTGRES_CONTENT_URL:+--postgres-content-url=\"$POSTGRES_CONTENT_URL\"} ${ELASTICSEARCH_URL:+--elasticsearch-node-url=\"$ELASTICSEARCH_URL\"} ${ELASTICSEARCH_PASSWORD:+--elasticsearch-basic-auth=\"elastic:$ELASTICSEARCH_PASSWORD\"} ${ES_OBJECTS_URL:+--es-objects-elasticsearch-url=\"$ES_OBJECTS_URL\"} ${ELASTICSEARCH_PASSWORD:+--es-objects-auth=\"elastic:$ELASTICSEARCH_PASSWORD\"} ${POSTGRES_INDEXER_URL:+--postgres-indexer-url=\"$POSTGRES_INDEXER_URL\"}"),
								pulumi.String("--"),
							},
							Args: func() pulumi.StringArray {
								baseArgs := pulumi.StringArray{
									pulumi.String("--data-dir=/data"),
									pulumi.String("--p2p-endpoint=0.0.0.0:2771"),
								}

								if args.GenesisURL != nil {
									baseArgs = append(baseArgs, pulumi.String("--genesis-json=/genesis/genesis.json"))
								}

								if args.SeedNodes != nil {
									seedNodesArg := args.SeedNodes.ToStringArrayOutput().ApplyT(func(nodes []string) string {
										nodeList := "[]"
										if len(nodes) > 0 {
											nodeList = fmt.Sprintf(`["%s"]`, strings.Join(nodes, `" "`))
										}
										return fmt.Sprintf("--seed-nodes=%s", nodeList)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, seedNodesArg)
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

								if args.PostgresStartBlock != nil {
									startBlockArg := args.PostgresStartBlock.ToIntOutput().ApplyT(func(block int) string {
										return fmt.Sprintf("--postgres-content-start-block=%d", block)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, startBlockArg)
								}

								if args.ElasticsearchIndexPrefix != nil {
									indexPrefixArg := args.ElasticsearchIndexPrefix.ToStringOutput().ApplyT(func(prefix string) string {
										return fmt.Sprintf("--elasticsearch-index-prefix=%s", prefix)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, indexPrefixArg)

									esObjectsIndexPrefixArg := args.ElasticsearchIndexPrefix.ToStringOutput().ApplyT(func(prefix string) string {
										return fmt.Sprintf("--es-objects-index-prefix=%s", prefix)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, esObjectsIndexPrefixArg)
								}

								if args.ElasticsearchStartBlock != nil {
									esStartBlockArg := args.ElasticsearchStartBlock.ToIntOutput().ApplyT(func(block int) string {
										return fmt.Sprintf("--elasticsearch-start-es-after-block=%d", block)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, esStartBlockArg)

									esObjectsStartBlockArg := args.ElasticsearchStartBlock.ToIntOutput().ApplyT(func(block int) string {
										return fmt.Sprintf("--es-objects-start-es-after-block=%d", block)
									}).(pulumi.StringOutput)

									baseArgs = append(baseArgs, esObjectsStartBlockArg)
								}

								if args.PostgresIndexerMode != nil {
									modeArg := args.PostgresIndexerMode.ToIntOutput().ApplyT(func(mode int) string {
										return fmt.Sprintf("--postgres-indexer-mode=%d", mode)
									}).(pulumi.StringOutput)
									baseArgs = append(baseArgs, modeArg)
								}

								if args.PostgresIndexerStartBlock != nil {
									startBlockArg := args.PostgresIndexerStartBlock.ToIntOutput().ApplyT(func(block int) string {
										return fmt.Sprintf("--postgres-indexer-start-after-block=%d", block)
									}).(pulumi.StringOutput)
									baseArgs = append(baseArgs, startBlockArg)
								}

								if args.PostgresIndexerVisitor != nil {
									visitorArg := args.PostgresIndexerVisitor.ToBoolOutput().ApplyT(func(v bool) string {
										return fmt.Sprintf("--postgres-indexer-visitor=%t", v)
									}).(pulumi.StringOutput)
									baseArgs = append(baseArgs, visitorArg)
								}

								if args.PostgresIndexerOperationString != nil {
									opStringArg := args.PostgresIndexerOperationString.ToBoolOutput().ApplyT(func(v bool) string {
										return fmt.Sprintf("--postgres-indexer-operation-string=%t", v)
									}).(pulumi.StringOutput)
									baseArgs = append(baseArgs, opStringArg)
								}

								return baseArgs
							}(),
							Env: func() corev1.EnvVarArray {
								envVars := corev1.EnvVarArray{}

								if args.PostgresSecretName != nil && args.PostgresSecretKey != nil {
									envVars = append(envVars, &corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_CONTENT_URL"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: args.PostgresSecretName.ToStringOutput(),
												Key:  args.PostgresSecretKey.ToStringOutput(),
											},
										},
									})
								}

								if args.ElasticsearchURL != nil {
									envVars = append(envVars, &corev1.EnvVarArgs{
										Name:  pulumi.String("ELASTICSEARCH_URL"),
										Value: args.ElasticsearchURL,
									})
									envVars = append(envVars, &corev1.EnvVarArgs{
										Name:  pulumi.String("ES_OBJECTS_URL"),
										Value: args.ElasticsearchURL,
									})
								}

								if args.ElasticsearchSecretName != nil {
									envVars = append(envVars, &corev1.EnvVarArgs{
										Name: pulumi.String("ELASTICSEARCH_PASSWORD"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: args.ElasticsearchSecretName.ToStringOutput(),
												Key:  pulumi.String("elastic"),
											},
										},
									})
								}

								if args.PostgresIndexerSecretName != nil && args.PostgresIndexerSecretKey != nil {
									envVars = append(envVars, &corev1.EnvVarArgs{
										Name: pulumi.String("POSTGRES_INDEXER_URL"),
										ValueFrom: &corev1.EnvVarSourceArgs{
											SecretKeyRef: &corev1.SecretKeySelectorArgs{
												Name: args.PostgresIndexerSecretName.ToStringOutput(),
												Key:  args.PostgresIndexerSecretKey.ToStringOutput(),
											},
										},
									})
								}

								return envVars
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
		pulumi.DependsOn([]pulumi.Resource{NodePVC, GenesisPVC}),
	)
	if err != nil {
		return nil, err
	}

	component.Deployment = Deployment

	return component, nil
}
