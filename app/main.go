package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Cluster struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Project struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ClusterID   string            `json:"clusterId"`
	Annotations map[string]string `json:"annotations"`
}

func main() {
	rancherAPIURL := os.Getenv("RANCHER_SERVER_URL")
	accessToken := os.Getenv("RANCHER_TOKEN_KEY")

	//time.Sleep(5 * time.Minute)

	clusters := getClusters(rancherAPIURL, accessToken)
	configMapData := make(map[string]string)

	for _, cluster := range clusters {
		if cluster.Type == "cluster" {
			clusterData := fmt.Sprintf("Cluster ID: %s, Name: %s", cluster.ID, cluster.Name)
			configMapData[cluster.ID] = clusterData

			projects := getProjects(rancherAPIURL, accessToken, cluster.ID)
			for _, project := range projects {
				projectData := fmt.Sprintf("Project ID: %s, Name: %s", project.ID, project.Name)
				for key, value := range project.Annotations {
					projectData += fmt.Sprintf(", Annotation: %s = %s", key, value)
				}
				configMapData[project.ID] = projectData
			}
		}
	}

	updateConfigMap(configMapData)
}

func getKubeClient() (*kubernetes.Clientset, error) {
	log.Println("Starting getKubeClient function")

	// Create config. In-cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error creating in-cluster config: %v", err)
		return nil, err
	}

	// Create a Clientset using the config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes clientset: %v", err)
		return nil, err
	}

	log.Println("Successfully initialized Kubernetes clientset")
	return clientset, nil
}

func updateConfigMap(data map[string]string) {
	log.Println("Starting updateConfigMap function")

	clientset, err := getKubeClient()
	if err != nil {
		log.Fatalf("Error getting Kube client: %v", err)
		return
	}

	cmClient := clientset.CoreV1().ConfigMaps("kube-system")

	cm, err := cmClient.Get(context.TODO(), "rancher-data", metav1.GetOptions{})
	if err != nil {
		log.Println("ConfigMap 'rancher-data' not found, attempting to create")

		// If it doesn't exist, create it
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rancher-data",
			},
			Data: map[string]string{
				"clusters": "",
				"projects": "",
			},
		}
		_, err = cmClient.Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("Error creating ConfigMap: %v", err)
			return
		}
		log.Println("Successfully created ConfigMap 'rancher-data'")
	} else {
		// If it exists, update it
		log.Println("ConfigMap 'rancher-data' found, updating")
		cm.Data["clusters"] = ""
		cm.Data["projects"] = ""
		_, err = cmClient.Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			log.Fatalf("Error updating ConfigMap: %v", err)
			return
		}
		log.Println("Successfully updated ConfigMap 'rancher-data'")
	}

	// Iterate over the clusters and add them to the configmap
	for clusterID, clusterData := range data {
		cm.Data["clusters"] += fmt.Sprintf("---\n%s\n", clusterID+":"+clusterData)
	}

	// Iterate over the projects and add them to the configmap
	for projectID, projectData := range data {
		cm.Data["projects"] += fmt.Sprintf("---\n%s\n", projectID+":"+projectData)
	}

	_, err = cmClient.Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		log.Fatalf("Error updating ConfigMap: %v", err)
		return
	}
	log.Println("Successfully updated ConfigMap 'rancher-data'")
}

func getHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

func getClusters(rancherAPIURL string, accessToken string) []Cluster {
	log.Println("Starting getClusters function")

	client := getHttpClient()
	req, err := http.NewRequest("GET", rancherAPIURL+"/clusters", nil)
	if err != nil {
		log.Fatalf("Error creating new request to Rancher API: %v", err)
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request to Rancher API: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code from Rancher API: %d\n", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body from Rancher API: %v", err)
		return nil
	}

	var response struct {
		Data []Cluster `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalf("Error unmarshaling response body: %v", err)
		return nil
	}

	log.Printf("Fetched %d clusters from Rancher API", len(response.Data))
	return response.Data
}

func getProjects(rancherAPIURL string, accessToken string, clusterID string) []Project {
	log.Printf("Starting getProjects function for cluster ID: %s", clusterID)

	client := getHttpClient()
	req, err := http.NewRequest("GET", rancherAPIURL+"/projects?clusterId="+clusterID, nil)
	if err != nil {
		log.Fatalf("Error creating new request to Rancher API for projects: %v", err)
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request to Rancher API for projects: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code from Rancher API for projects: %d\n", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body from Rancher API for projects: %v", err)
		return nil
	}

	var response struct {
		Data []Project `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalf("Error unmarshaling response body for projects: %v", err)
		return nil
	}

	log.Printf("Fetched %d projects for cluster ID %s from Rancher API", len(response.Data), clusterID)
	return response.Data
}
