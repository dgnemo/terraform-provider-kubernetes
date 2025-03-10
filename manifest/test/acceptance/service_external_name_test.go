//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

// This test case tests a Service but also is a demonstration of some the assert functions
// available in the test helper
func TestKubernetesManifest_Service_ExternalName(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "services", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Service_ExternalName/service.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "services", namespace, name)

	s, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.spec.selector.app":  "test",
		"kubernetes_manifest.test.object.spec.type":          "ExternalName",
		"kubernetes_manifest.test.object.spec.externalName":  "terraform.kubernetes.test.com",
	})

	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.object.spec.ports.0")

	tfconfigModified := loadTerraformConfig(t, "Service_ExternalName/service_modified.tf", tfvars)
	tf.SetConfig(ctx, tfconfigModified)
	tf.Apply(ctx)

	s, err = tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":        namespace,
		"kubernetes_manifest.test.object.metadata.name":             name,
		"kubernetes_manifest.test.object.metadata.annotations.test": "1",
		"kubernetes_manifest.test.object.metadata.labels.test":      "2",
		"kubernetes_manifest.test.object.spec.selector.app":         "test",
		"kubernetes_manifest.test.object.spec.type":                 "ExternalName",
		"kubernetes_manifest.test.object.spec.externalName":         "kubernetes-alpha.terraform.test.com",
	})

	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.labels", 1)
	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.annotations", 1)

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test.object.metadata.labels.test")

	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.spec")

}
