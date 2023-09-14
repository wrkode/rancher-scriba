package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

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

const maxRetries = 5

func exponentialBackoff(retry int) time.Duration {
	return time.Duration(math.Pow(2, float64(retry))) * time.Second
}

func withRetry(fn func() error) error {
	for i := 0; i <= maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		log.Printf("Error encountered: %v. Retrying in %v seconds", err, exponentialBackoff(i+1).Seconds())
		time.Sleep(exponentialBackoff(i + 1))
	}
	return fmt.Errorf("after %d retries, operation failed", maxRetries)
}

func main() {
	rancherAPIURL := os.Getenv("RANCHER_SERVER_URL")
	accessToken := os.Getenv("RANCHER_TOKEN_KEY")

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
	var clusters []Cluster

	err := withRetry(func() error {
		client := getHttpClient()
		req, err := http.NewRequest("GET", rancherAPIURL+"/clusters", nil)
		if err != nil {
			log.Printf("Error creating new request to Rancher API: %v", err)
			return err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending request to Rancher API: %v", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Unexpected status code from Rancher API: %d\n", resp.StatusCode)
			return fmt.Errorf("Unexpected status code from Rancher API: %d", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body from Rancher API: %v", err)
			return err
		}

		var response struct {
			Data []Cluster `json:"data"`
		}
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Printf("Error unmarshaling response body: %v", err)
			return err
		}

		clusters = response.Data

		log.Printf("Fetched %d clusters from Rancher API", len(response.Data))
		return nil // No error, so returning nil
	})

	if err != nil {
		log.Fatalf("Failed to fetch clusters after retries: %v", err)
		return nil
	}

	return clusters
}

func getProjects(rancherAPIURL string, accessToken string, clusterID string) []Project {
	log.Printf("Starting getProjects function for cluster ID: %s", clusterID)
	var projects []Project

	err := withRetry(func() error {
		client := getHttpClient()
		req, err := http.NewRequest("GET", rancherAPIURL+"/projects?clusterId="+clusterID, nil)
		if err != nil {
			log.Printf("Error creating new request to Rancher API for projects: %v", err)
			return err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending request to Rancher API for projects: %v", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("unexpected status code from Rancher API for projects: %d\n", resp.StatusCode)
			return fmt.Errorf("unexpected status code from Rancher API for projects: %d", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body from Rancher API for projects: %v", err)
			return err
		}

		var response struct {
			Data []Project `json:"data"`
		}
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Printf("Error unmarshaling response body for projects: %v", err)
			return err
		}

		projects = response.Data

		log.Printf("Fetched %d projects for cluster ID %s from Rancher API", len(response.Data), clusterID)
		return nil // No error, so returning nil
	})

	if err != nil {
		log.Fatalf("Failed to fetch projects after retries: %v", err)
		return nil
	}

	return projects
}
