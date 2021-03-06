// Note: the example only works with the code within the same release/branch.
package gclustercore

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	uuid "github.com/satori/go.uuid"

	batchv1 "k8s.io/api/batch/v1"
	//appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	//"k8s.io/client-go/util/retry"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type TestConfiguration struct {
	GitRepo        string `json:"git_repo,omitempty"`
	Revision       string `json:"revision,omitempty"`
	SimulationName string `json:"simulation_name,omitempty"`
	Data           string `json:"data,omitempty"`
	NbInjectords   int32  `json:"nb_injectors,omitempty"`
	NbVirtualUsers int32  `json:"nb_vu,omitempty"`
	Duration       int64  `json:"duration,omitempty"`
	Ramp           int64  `json:"ramp,omitempty"`
}

type Configuration struct {
	PerfImage    string
	WatcherImage string
}

type Namespace int

const (
	NamespaceDev Namespace = iota
	NamespaceValid
	NamespaceProd
)

func (s Namespace) String() string {
	return [...]string{"dev", "valid", "prod"}[s]
}

func getKubeClient() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return kubeClient
}

func getJobsClient(kubeClient *kubernetes.Clientset, nameSpace Namespace) v1.JobInterface {
	jobsClient := kubeClient.BatchV1().Jobs(nameSpace.String())
	return jobsClient
}

func getPodInterface(kubeClient *kubernetes.Clientset, nameSpace Namespace) corev1.PodInterface {
	podInterface := kubeClient.CoreV1().Pods(nameSpace.String())
	return podInterface
}

var chars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func readConfiguration() Configuration {
	fmt.Println("Read configuration conf.json")
	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		panic(err)
	}
	return configuration
}

func LaunchTest(t *TestConfiguration, nameSpace Namespace) string {

	configuration := readConfiguration()
	fmt.Println(configuration.PerfImage)

	rand.Seed(time.Now().UnixNano())
	testId := randSeq(5)

	kubeClient := getKubeClient()
	jobsClient := getJobsClient(kubeClient, nameSpace)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "batch-job-" + testId,
			Namespace: nameSpace.String(),
			Labels:    map[string]string{"type": "batch-job", "simulation_id": testId},
		},
		Spec: batchv1.JobSpec{
			Parallelism:           int32Ptr(t.NbInjectords),
			Completions:           int32Ptr(t.NbInjectords),
			BackoffLimit:          int32Ptr(1),
			ActiveDeadlineSeconds: &(&struct{ x int64 }{t.Duration + 3600}).x,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"type": "batch-job-pod", "simulation_id": testId},
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						{
							Name: "nfs",
							VolumeSource: apiv1.VolumeSource{
								PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
									ClaimName: "nfs",
								},
							},
						},
						{
							Name: "git-repo",
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []apiv1.Container{
						{
							Name:    "prepare-test",
							Image:   "busybox",
							Command: []string{"sh", "-c", "mkdir -p /exports/results/" + testId + ";"},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "nfs",
									MountPath: "/exports",
								},
							},
						},
						{
							Name:  "git-clone",
							Image: "alpine/git",
							Args:  []string{"clone", "--single-branch", "--branch", t.Revision, t.GitRepo, "/repo"},
							Env: []apiv1.EnvVar{
								{
									Name: "GIT_USERNAME",
									ValueFrom: &apiv1.EnvVarSource{
										SecretKeyRef: &apiv1.SecretKeySelector{
											LocalObjectReference: apiv1.LocalObjectReference{Name: "git-secret"},
											Key:                  "username",
										},
									},
								},
								{
									Name: "GIT_PASSWORD",
									ValueFrom: &apiv1.EnvVarSource{
										SecretKeyRef: &apiv1.SecretKeySelector{
											LocalObjectReference: apiv1.LocalObjectReference{Name: "git-secret"},
											Key:                  "password",
										},
									},
								},
							},

							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "git-repo",
									MountPath: "/repo",
								},
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name:  "main",
							Image: configuration.PerfImage,
							Env: []apiv1.EnvVar{
								{
									Name:  "SIMULATION_NAME",
									Value: t.SimulationName,
								},
								{
									Name:  "DATA",
									Value: t.Data,
								},
								{
									Name:  "NBUSERS",
									Value: fmt.Sprint(t.NbVirtualUsers),
								},
								{
									Name:  "RAMP",
									Value: t.SimulationName,
								},
								{
									Name:  "DURATION",
									Value: fmt.Sprint(t.Duration),
								},
								{
									Name:  "SIMULATION_ID",
									Value: testId,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "nfs",
									MountPath: "/gatling-charts-highcharts-bundle-3.0.2/results",
									SubPath:   "results/" + testId,
								},
								{
									Name:      "git-repo",
									MountPath: "/gatling-charts-highcharts-bundle-3.0.2/user-files",
									ReadOnly:  true,
								},
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}

	// Create Job
	fmt.Println("Creating job...")
	job_result, job_err := jobsClient.Create(job)
	if job_err != nil {
		panic(job_err)
	}
	fmt.Printf("Created job %q.\n", job_result.GetObjectMeta().GetName())

	job_watcher := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "batch-watcher-" + testId,
			Namespace: nameSpace.String(),
			Labels:    map[string]string{"type": "batch-watcher", "simulation_id": testId},
		},
		Spec: batchv1.JobSpec{
			Parallelism:           int32Ptr(1),
			Completions:           int32Ptr(1),
			BackoffLimit:          int32Ptr(0),
			ActiveDeadlineSeconds: &(&struct{ x int64 }{t.Duration + 3600}).x,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"type": "batch-watcher-pod", "simulation_id": testId},
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						{
							Name: "nfs",
							VolumeSource: apiv1.VolumeSource{
								PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
									ClaimName: "nfs",
								},
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name:  "watcher",
							Image: configuration.WatcherImage,
							Env: []apiv1.EnvVar{
								{
									Name:  "DURATION",
									Value: fmt.Sprint(t.Duration),
								},
								{
									Name:  "SIMULATION_ID",
									Value: testId,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "nfs",
									MountPath: "/results",
									SubPath:   "results/" + testId,
								},
								{
									Name:      "nfs",
									MountPath: "/aggregated-reports",
								},
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}

	// Create Job Watcher
	fmt.Println("Creating job watcher...")
	job_watcher_result, job_watcher_err := jobsClient.Create(job_watcher)
	if job_watcher_err != nil {
		panic(job_watcher_err)
	}
	fmt.Printf("Created job watcher %q.\n", job_watcher_result.GetObjectMeta().GetName())

	return testId

}

func GetStatus(id *uuid.UUID, nameSpace Namespace) *batchv1.JobStatus {
	//kubectl get job batch-job --output json
	kubeClient := getKubeClient()
	jobsClient := getJobsClient(kubeClient, nameSpace)
	job, err := jobsClient.Get("batch-job", metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	return &job.Status
}

func deletePods(kubeClient *kubernetes.Clientset, nameSpace Namespace, s string) {
	podInterface := getPodInterface(kubeClient, nameSpace)
	podList, err := podInterface.List(metav1.ListOptions{
		LabelSelector: s,
	})

	if err != nil {
		panic(err)
	}

	for _, pod := range podList.Items {
		err = podInterface.Delete(pod.Name, &metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
	}
}

func DeleteJobs(nameSpace Namespace) {
	kubeClient := getKubeClient()
	jobsClient := getJobsClient(kubeClient, nameSpace)
	jobsList, _ := jobsClient.List(metav1.ListOptions{})

	if len(jobsList.Items) > 0 {
		for _, job := range jobsList.Items {
			err := jobsClient.Delete(job.Name, &metav1.DeleteOptions{})
			if err != nil {
				panic(err)
			}
		}
	}

	deletePods(kubeClient, nameSpace, "type=batch-watcher-pod")
	deletePods(kubeClient, nameSpace, "type=batch-job-pod")
}

func int32Ptr(i int32) *int32 { return &i }
