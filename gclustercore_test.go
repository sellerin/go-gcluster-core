package gclustercore

import (
	"fmt"
	"testing"
	"encoding/json"
	uuid "github.com/satori/go.uuid"
)

func TestLaunch(t *testing.T) {

	tconf := &TestConfiguration{
		GitRepo:        "https://github.com/sellerin/gatling-cluster.git",
		Revision:       "master",
		SimulationName: "c2gwebaws.C2gwebSimulation",
		NbInjectords:   2,
		NbVirtualUsers: 2,
		Duration:       300,
	}

	id := LaunchTest(tconf)
	fmt.Printf("Gatling test started. Id: %s\n", id)

}

func TestGetStatus(t *testing.T) {

	id, _ := uuid.FromString("e9b728fd-5f50-46d3-bd5f-db0f82a0c711")

	status := GetStatus(&id)

	status2B, _ := json.Marshal(*status)
    fmt.Println(string(status2B))

}

func TestDeleteJobs(t *testing.T) {
    DeleteJobs()
}


	/*
		deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "demo-deployment",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "demo",
					},
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:  "web",
								Image: "nginx:1.12",
								Ports: []apiv1.ContainerPort{
									{
										Name:          "http",
										Protocol:      apiv1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}

		// Create Deployment
		fmt.Println("Creating deployment...")
		result, err := deploymentsClient.Create(deployment)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

		// Update Deployment
		prompt()
		fmt.Println("Updating deployment...")
		//    You have two options to Update() this Deployment:
		//
		//    1. Modify the "deployment" variable and call: Update(deployment).
		//       This works like the "kubectl replace" command and it overwrites/loses changes
		//       made by other clients between you Create() and Update() the object.
		//    2. Modify the "result" returned by Get() and retry Update(result) until
		//       you no longer get a conflict error. This way, you can preserve changes made
		//       by other clients between Create() and Update(). This is implemented below
		//			 using the retry utility package included with client-go. (RECOMMENDED)
		//
		// More Info:
		// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Retrieve the latest version of Deployment before attempting update
			// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
			result, getErr := deploymentsClient.Get("demo-deployment", metav1.GetOptions{})
			if getErr != nil {
				panic(fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
			}

			result.Spec.Replicas = int32Ptr(1)                           // reduce replica count
			result.Spec.Template.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
			_, updateErr := deploymentsClient.Update(result)
			return updateErr
		})
		if retryErr != nil {
			panic(fmt.Errorf("Update failed: %v", retryErr))
		}
		fmt.Println("Updated deployment...")

		// List Deployments
		prompt()
		fmt.Printf("Listing deployments in namespace %q:\n", apiv1.NamespaceDefault)
		list, err := deploymentsClient.List(metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, d := range list.Items {
			fmt.Printf(" * %s (%d replicas)\n", d.Name, *d.Spec.Replicas)
		}

		// Delete Deployment
		prompt()
		fmt.Println("Deleting deployment...")
		deletePolicy := metav1.DeletePropagationForeground
		if err := deploymentsClient.Delete("demo-deployment", &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			panic(err)
		}
		fmt.Println("Deleted deployment.")
	*/


/*

import (
	"bufio"
	"os"
)

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}*/