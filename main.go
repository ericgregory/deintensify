package main

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

type GCPRegion struct {
	name string
	intensity int
}

func main() {

	// For the sake of time, hard-coding GCP's grid carbon intensity (gCO2eq/kWh) by region. 
	// This data from Google is the data Cloud Carbon Footprint uses.

	gcpregions := []GCPRegion{
		{"asia-east1", 456}, 
		{"asia-east2", 360}, 
		{"asia-northeast1", 464}, 
		{"asia-northeast2", 384},
		{"asia-northeast3", 425},
		{"asia-south1", 670},
		{"asia-south2", 671},
		{"asia-southeast1", 372},
		{"asia-southeast2", 580},
		{"australia-southeast1", 598},
		{"australia-southeast2", 521},
		{"europe-central2", 576},
		{"europe-north1", 127},
		{"europe-southwest1", 121},
		{"europe-west1", 110},
		{"europe-west2", 172},
		{"europe-west3", 269},
		{"europe-west4", 283},
		{"europe-west6", 86},
		{"europe-west8", 298},
		{"europe-west9", 59},
		{"northamerica-northeast1", 0},
		{"northamerica-northeast2", 29},
		{"southamerica-east1", 129},
		{"southamerica-west1", 190},
		{"us-central1", 394},
		{"us-east1", 434},
		{"us-east4", 309},
		{"us-east5", 309},
		{"us-south1", 296},
		{"us-west1", 60},
		{"us-west2", 190},
		{"us-west3", 448},
		{"us-west4", 365}}
	
	// Sorting the regions by intensity, from lowest to highest.
	
	sort.SliceStable(gcpregions, func(i, j int) bool {
		return gcpregions[i].intensity < gcpregions[j].intensity
	})
	
	// Capturing the least-intensity target region in the target variable.

	target := gcpregions[0].name
	
	// Here we're setting variables to hold information about our management cluster environment.

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
			fmt.Println("Changing region from", region, "to", target)
			// Update the region spec to target
			unstructured.SetNestedField(item.Object, target, "spec", "region")
		}
	}
}

// Since we're using Cluster API's custom resources for clusters (in this case the GCPCluster resource), 
// we're interacting with the resource dynamically/generically rather than through typed/pre-defined interfaces.

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