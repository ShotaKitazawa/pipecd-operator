package controllers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func generatePatchFinalizersObject(gvk schema.GroupVersionKind, namespace, name string, finalizers []string) *unstructured.Unstructured {
	patch := &unstructured.Unstructured{}
	patch.SetGroupVersionKind(gvk)
	patch.SetNamespace(namespace)
	patch.SetName(name)
	patch.UnstructuredContent()["objectMeta"] = map[string]interface{}{
		"finalizers": finalizers,
	}
	return patch
}
