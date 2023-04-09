package main

import (
	"context"
	"fmt"
//	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	ctx := context.Background()
	config := ctrl.GetConfigOrDie()
	dynamic := dynamic.NewForConfigOrDie(config)
	namespace := "default"

	items, err := GetResourcesDynamically(dynamic, ctx, "infrastructure.cluster.x-k8s.io", "v1beta1", "gcpclusters", namespace)
// Example using configMaps...	items, err := GetResourcesDynamically(dynamic, ctx, "", "v1", "configmaps", namespace)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, item := range items {
			// fmt.Printf("%+v\n", item)
			fmt.Println(item.GetName())
			region, _, _ := unstructured.NestedString(item.Object, "spec", "region")
			fmt.Println(region)
		}
	}
}

// Since we're using Cluster API's custom resources for clusters (in this case the GCPCluster resource), we're interacting with the resource dynamically/generically rather than through typed/pre-defined interfaces.

func GetResourcesDynamically(dynamic dynamic.Interface, ctx context.Context, group string, version string, resource string, namespace string) (
	[]unstructured.Unstructured, error) {

	resourceId := schema.GroupVersionResource{
		Group: group,
		Version:  version,
		Resource: resource,
	}
	list, err := dynamic.Resource(resourceId).Namespace(namespace).
		List(ctx, metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}