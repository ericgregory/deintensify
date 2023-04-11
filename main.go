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
	
	// Here we're setting variables to hold information about our management cluster environment and resource.

	ctx := context.Background()
	config := ctrl.GetConfigOrDie()
	dynamic := dynamic.NewForConfigOrDie(config)
	namespace := "default"
	res := schema.GroupVersionResource{Group: "infrastructure.cluster.x-k8s.io", Version: "v1beta1", Resource: "gcpclusters"}

	// Using the GetResourcesDynamically function to get a list of resources that we will loop through.

	items, err := GetResourcesDynamically(dynamic, ctx, "infrastructure.cluster.x-k8s.io", "v1beta1", "gcpclusters", namespace)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, item := range items {
			// Grab the region of the current resource in the loop

			region, _, _ := unstructured.NestedString(item.Object, "spec", "region")
			fmt.Println(item.GetName(), "is on region", region)

			// If the region doesn't match the target, we'll delete this cluster and create a new
			// version with the same specs in the target region

			if region != target {
				fmt.Println("Deleting cluster from", region, "and creating new cluster on", target)

				// Grab unique fields from current worker cluster

				genname, _, _ := unstructured.NestedString(item.Object, "metadata", "generateName")
				network, _, _ := unstructured.NestedString(item.Object, "spec", "network", "name")
				project, _, _ := unstructured.NestedString(item.Object, "spec", "project")

				// Here we're defining the GCPCluster object

				desired := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "infrastructure.cluster.x-k8s.io/v1beta1",
						"kind":       "GCPCluster",
						"metadata": map[string]interface{}{
							"namespace":    namespace,
							"generateName": genname,
						},
						"spec": map[string]interface{}{
							"network": map[string]interface{}{
								"name": network,
							},
						"project": project,
						"region": target,
						},
					},
				}

				// Create new cluster with same values in target region

				created, err := dynamic.
					Resource(res).
					Namespace(namespace).
					Create(ctx, desired, metav1.CreateOptions{})
				if err != nil {
					panic(err.Error())
				}
				fmt.Println(created.GetName(), "created") 

				// Delete old cluster

				err = dynamic.
					Resource(res).
					Namespace(namespace).
					Delete(
						ctx,
						item.GetName(),
						metav1.DeleteOptions{},
					)
				if err != nil {
					panic(err.Error())
				} else {
					fmt.Println(item.GetName(), "deleted")
				}

			}	
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

