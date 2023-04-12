#deintensify

This is a simple sample app intended to demonstrate how you can build on existing open source projects to create emissions-aware Kubernetes infrastructure.  

The app supposes that you are managing worker clusters via Cluster API on GCP, and that you would like to programmatically assign GCP worker clusters to less carbon-intense regions. It grabs emissions data from the Cloud Carbon Footprint API and reassigns worker cluster objects accordingly.  

To run deintensify, you will need...

- [minikube](https://minikube.sigs.k8s.io/docs/start/) (or your own preferred flavor of local cluster)
    - configured as a [Cluster API](https://cluster-api.sigs.k8s.io/user/quick-start.html) management cluster with the gcp provider
- [Go](https://go.dev/) v1.20.1 or higher
- [Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/docs/getting-started) running locally on default ports (recommend mocked data)

Before running, create some arbitrary gcpcluster resources with the included cluster-demo.yaml:

    % kubectl create -f demo-cluster.yaml
