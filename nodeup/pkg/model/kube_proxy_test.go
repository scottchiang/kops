/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"reflect"
	"testing"

	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/flagbuilder"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/util/pkg/architectures"
	"k8s.io/kops/util/pkg/exec"

	"github.com/blang/semver/v4"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKubeProxyBuilder_buildPod(t *testing.T) {
	// kube proxy spec can be found here.
	// https://pkg.go.dev/k8s.io/kops/pkg/apis/kops#ClusterSpec
	// https://pkg.go.dev/k8s.io/kops/pkg/apis/kops#KubeProxyConfig

	cluster := &kops.Cluster{}
	cluster.Spec.MasterInternalName = "dev-cluster"

	cluster.Spec.KubeProxy = &kops.KubeProxyConfig{}
	cluster.Spec.KubeProxy.Image = "kube-proxy:1.2"
	cluster.Spec.KubeProxy.CPURequest = resource.NewScaledQuantity(20, resource.Milli)
	cluster.Spec.KubeProxy.CPULimit = resource.NewScaledQuantity(30, resource.Milli)

	flags, _ := flagbuilder.BuildFlagsList(cluster.Spec.KubeProxy)

	flags = append(flags, []string{
		"--kubeconfig=/var/lib/kube-proxy/kubeconfig",
		"--oom-score-adj=-998",
		`--resource-container=""`,
	}...)

	type fields struct {
		NodeupModelContext *NodeupModelContext
	}
	tests := []struct {
		name    string
		fields  fields
		want    *v1.Pod
		wantErr bool
	}{
		{
			"Setup KubeProxy for kubernetes version 1.20",
			fields{
				&NodeupModelContext{
					Cluster:           cluster,
					kubernetesVersion: semver.Version{Major: 1, Minor: 20},
				},
			},
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kube-proxy",
					Namespace: "kube-system",
					Labels: map[string]string{
						"k8s-app": "kube-proxy",
						"tier":    "node",
					},
				},
				Spec: v1.PodSpec{
					HostNetwork: true,
					Containers: []v1.Container{
						{
							Name:  "kube-proxy",
							Image: "kube-proxy:1.2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU: *resource.NewScaledQuantity(20, resource.Milli),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU: *resource.NewScaledQuantity(30, resource.Milli),
								},
							},
							Command: exec.WithTee("/usr/local/bin/kube-proxy", sortedStrings(flags), "/var/log/kube-proxy.log"),
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &KubeProxyBuilder{
				NodeupModelContext: tt.fields.NodeupModelContext,
			}
			got, err := b.buildPod()
			if (err != nil) != tt.wantErr {
				t.Errorf("KubeProxyBuilder.buildPod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// compare object metadata Namespace
			if !reflect.DeepEqual(got.ObjectMeta.Namespace, tt.want.ObjectMeta.Namespace) {
				t.Errorf("KubeProxyBuilder.buildPod() ObjectMeta namespace = %v, want %v", got.ObjectMeta.Namespace, tt.want.ObjectMeta.Namespace)
			}

			// compare object metadata name
			if !reflect.DeepEqual(got.ObjectMeta.Name, tt.want.ObjectMeta.Name) {
				t.Errorf("KubeProxyBuilder.buildPod() ObjectMeta Name = %v, want %v", got.ObjectMeta.Name, tt.want.ObjectMeta.Name)
			}

			// compare object metadata Labels
			if !reflect.DeepEqual(got.ObjectMeta.Labels, tt.want.ObjectMeta.Labels) {
				t.Errorf("KubeProxyBuilder.buildPod() ObjectMeta Labels = %v, want %v", got.ObjectMeta.Labels, tt.want.ObjectMeta.Labels)
			}

			if len(got.Spec.Containers) != 1 {
				t.Errorf("KubeProxyBuilder.buildPod() expected exactly one container in kube-proxy but got %v", len(got.Spec.Containers))
			}

			// compare pod spec container name
			if !reflect.DeepEqual(got.Spec.Containers[0].Name, tt.want.Spec.Containers[0].Name) {
				t.Errorf("KubeProxyBuilder.buildPod() Container Name = %v, want %v", got.Spec.Containers[0].Name, tt.want.Spec.Containers[0].Name)
			}
			// compare pod spec container Image
			if !reflect.DeepEqual(got.Spec.Containers[0].Image, tt.want.Spec.Containers[0].Image) {
				t.Errorf("KubeProxyBuilder.buildPod() Image Name = %v, want %v", got.Spec.Containers[0].Image, tt.want.Spec.Containers[0].Image)
			}

			// compare pod spec container resource
			if !reflect.DeepEqual(got.Spec.Containers[0].Resources, tt.want.Spec.Containers[0].Resources) {
				t.Errorf("KubeProxyBuilder.buildPod() Resources = %v, want %v", got.Spec.Containers[0].Resources, tt.want.Spec.Containers[0].Resources)
			}
		})
	}
}

func TestKubeProxyBuilder(t *testing.T) {
	RunGoldenTest(t, "tests/golden/minimal", "kube-proxy", func(nodeupModelContext *NodeupModelContext, target *fi.ModelBuilderContext) error {
		builder := KubeProxyBuilder{NodeupModelContext: nodeupModelContext}
		return builder.Build(target)
	})
}

func TestKubeProxyBuilderAMD64(t *testing.T) {
	RunGoldenTest(t, "tests/golden/side-loading", "kube-proxy-amd64", func(nodeupModelContext *NodeupModelContext, target *fi.ModelBuilderContext) error {
		builder := KubeProxyBuilder{NodeupModelContext: nodeupModelContext}
		return builder.Build(target)
	})
}

func TestKubeProxyBuilderARM64(t *testing.T) {
	RunGoldenTest(t, "tests/golden/side-loading", "kube-proxy-arm64", func(nodeupModelContext *NodeupModelContext, target *fi.ModelBuilderContext) error {
		builder := KubeProxyBuilder{NodeupModelContext: nodeupModelContext}
		builder.Architecture = architectures.ArchitectureArm64
		return builder.Build(target)
	})
}
