package main

import (
	"context"
	"fmt"
	"sort"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Defining a struct to hold data from the Cloud Carbon Footprint API

type CCFRegion struct {
	Name      string  `json:"region"`
	Intensity float64 `json:"mtPerKwHour"`
}

// Logic to sort our structs according to emissions data later

type By func(p1, p2 *CCFRegion) bool

func (by By) Sort(GCPregions []CCFRegion) {
	ps := &regionSorter{
		GCPregions: GCPregions,
		by:      by, 
	}
	sort.Sort(ps)
}

type regionSorter struct {
	GCPregions []CCFRegion
	by      func(p1, p2 *CCFRegion) bool 
}

func (s *regionSorter) Len() int {
	return len(s.GCPregions)
}

func (s *regionSorter) Swap(i, j int) {
	s.GCPregions[i], s.GCPregions[j] = s.GCPregions[j], s.GCPregions[i]
}

func (s *regionSorter) Less(i, j int) bool {
	return s.by(&s.GCPregions[i], &s.GCPregions[j])
}

func main() {

	/* If you want to skip using the Cloud Carbon Footprint API, uncomment this to use hard-coded values 
	for GCP's grid carbon intensity (gCO2eq/kWh) by region from Google. Comment out the sections below on 
	"Grabbing emissions data..." and "Creating region structs from the JSON..." 
	
	GCPregions := []CCFRegion{
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
	
	*/
	
	// Grabbing emissions data from the Cloud Carbon Footprint API, running locally

	url := "http://localhost:4000/emissions"

	ccfClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "deintensify-sample")

	res, getErr := ccfClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	// Creating region structs with the JSON from the API

	var GCPregions = []CCFRegion{}
	jsonErr := json.Unmarshal(body, &GCPregions)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	
	// Sorting the regions by intensity, from lowest to highest.
	
	carbonintensity := func(p1, p2 *CCFRegion) bool {
		return p1.Intensity < p2.Intensity
	}

	By(carbonintensity).Sort(GCPregions)
	
	/* Capturing the least-intensity target region in the target variable. Note: Cloud Carbon Footprint is returning region data
	for multiple providers, so I'm being a little cheeky here and indexing straight to the lowest GCP region. Of course you'd just
	index to zero for the lowest overall value.
	*/

	target := GCPregions[7].Name
	
	// Here we're setting variables to hold information about our management cluster environment and resource.

	ctx := context.Background()
	config := ctrl.GetConfigOrDie()
	dynamic := dynamic.NewForConfigOrDie(config)
	namespace := "default"
	resourceId := schema.GroupVersionResource{Group: "infrastructure.cluster.x-k8s.io", Version: "v1beta1", Resource: "gcpclusters"}

	// Using the GetResourcesDynamically function to get a list of resources that we will loop through.

	items, err := GetResourcesDynamically(dynamic, ctx, "infrastructure.cluster.x-k8s.io", "v1beta1", "gcpclusters", namespace)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, item := range items {

			// Grab the region of the current resource in the loop

			region, _, _ := unstructured.NestedString(item.Object, "spec", "region")
			fmt.Println(item.GetName(), "is on region", region)

			/* If the region doesn't match the target, we'll delete this cluster and create a new
			   version with the same specs in the target region
			*/

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
					Resource(resourceId).
					Namespace(namespace).
					Create(ctx, desired, metav1.CreateOptions{})
				if err != nil {
					panic(err.Error())
				}
				fmt.Println(created.GetName(), "created") 

				// Delete old cluster

				err = dynamic.
					Resource(resourceId).
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

