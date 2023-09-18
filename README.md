# Rancher-Scriba

A Rancher Projects collector.
rancher-scriba is meant to be ran on downstream clusters. It connects to the Rancher Upstream API and will gather ```clusterID```, ```projectID```, and the annotations of each Rancher Project. The collected information will be stored in ```rancher-data``` ConfigMap in the ```kube-system``` namespace of the downstream cluster.
The ConfigMap can then be consumed by a Policy Engine.

## Requirements

The following preparation is required for rancher-scriba:

- rancher-scriba requires permission for CRUD operations of ConfigMap objects in the ```kube-system``` namespace.
For this, the ```sa_role_bindings.yaml``` file has been provided.
- An API Bearer Token needs to be created for rancher-scriba. Input this value into the ```secrets.sh``` file.
- The Rancher API endpoint to that rancher-scriba needs to connect to in to format ```https://RANCHER_FQDN>```. Input this value into the ```secrets.sh``` file.
- Adjust the collection interval (default 5 minutes) in ```rancher-cronjob.yaml```

## Deployment

This instructions assume that your ```kubeconfig``` context is set to the downstream cluster that will host rancher-scriba.

- While in the root of the repository, create ```rancher-api-secrets``` by running ```sh secrets.sh```.
- Create the role, role binding and service account with ```kubectl apply -f sa_role_bindings.yaml```.
- Create rancher-scriba cronjob in the ```kube-system``` namespace by running ```kubectl -n kube-system apply -f rancher-cronjob.yaml```.

If all actions are succesful, rancher-scriba will create a ConfigMap in the downstream cluster.
