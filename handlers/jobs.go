package handlers

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchtypev1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
)

var (
	trueVal  = true
	falseVal = false
)

var kubectlNamespace = os.Getenv("POD_NAMESPACE")

type JobsArray struct {
	JobInfo []JobInfo `json:"jobs"`
}

type JobInfo struct {
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type JobOutput struct {
	Output string `json:"output"`
}

func getJobClient() batchtypev1.JobInterface {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	// Access jobs. We can't do it all in one line, since we need to receive the
	// errors and manage thgem appropriately
	batchClient := clientset.BatchV1()
	jobsClient := batchClient.Jobs(kubectlNamespace)
	return jobsClient
}

func getJobByID(jc batchtypev1.JobInterface, jobid string) (*batchv1.Job, error) {
	jobs, err := jc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, job := range jobs.Items {
		if jobid == string(job.GetUID()) {
			return &job, nil
		}
	}
	return nil, fmt.Errorf("job with jobid %s not found", jobid)
}

func getJobStatusByID(jobid string) (*JobInfo, error) {
	job, err := getJobByID(getJobClient(), jobid)
	if err != nil {
		return nil, err
	}
	if job.Labels["app"] != "sowerjob" {
		return nil, fmt.Errorf("job with jobid %s not found", jobid)
	}
	ji := JobInfo{}
	ji.Name = job.Name
	ji.UID = string(job.GetUID())
	ji.Status = jobStatusToString(&job.Status)
	return &ji, nil
}

func listJobs(jc batchtypev1.JobInterface) JobsArray {
	jobs := JobsArray{}

	jobsList, err := jc.List(metav1.ListOptions{LabelSelector: "app=sowerjob"})

	if err != nil {
		return jobs
	}

	for _, job := range jobsList.Items {
		ji := JobInfo{}
		ji.Name = job.Name
		ji.UID = string(job.GetUID())
		ji.Status = jobStatusToString(&job.Status)
		jobs.JobInfo = append(jobs.JobInfo, ji)
	}

	return jobs
}

func jobStatusToString(status *batchv1.JobStatus) string {
	if status == nil {
		return "Unknown"
	}

	// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#jobstatus-v1-batch
	if status.Active >= 1 {
		return "Running"
	}
	if status.Succeeded >= 1 {
		return "Completed"
	}
	if status.Failed >= 1 {
		return "Failed"
	}
	return "Unknown"
}

func createK8sJob(inputData string, accessToken string, pelicanCreds PelicanCreds, peregrineCreds PeregrineCreds, userName string) (*JobInfo, error) {
	var conf = loadConfig("/sower_config.json")
	fmt.Println("config: ", conf)

	jobsClient := getJobClient()
	randname := GetRandString(5)
	name := fmt.Sprintf("%s-%s", conf.Name, randname)
	fmt.Println("input data: ", inputData)
	var deadline int64 = 3600
	var backoff int32 = 1
	labels := make(map[string]string)
	labels["app"] = "sowerjob"
	annotations := make(map[string]string)
	annotations["gen3username"] = userName

	var pullPolicies = map[string]k8sv1.PullPolicy{
		"always":         k8sv1.PullAlways,
		"if_not_present": k8sv1.PullIfNotPresent,
		"never":          k8sv1.PullNever,
	}

	var restartPolicies = map[string]k8sv1.RestartPolicy{
		"on_failure": k8sv1.RestartPolicyOnFailure,
		"never":      k8sv1.RestartPolicyNever,
	}

	var pullPolicy = k8sv1.PullPolicy(pullPolicies[conf.Container.PullPolicy])
	var restartPolicy = restartPolicies[conf.RestartPolicy]

	// For an example of how to create jobs, see this file:
	// https://github.com/pachyderm/pachyderm/blob/805e63/src/server/pps/server/api_server.go#L2320-L2345
	batchJob := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			// Optional: Parallelism:,
			// Optional: Completions:,
			// Optional: ActiveDeadlineSeconds:,
			// Optional: Selector:,
			// Optional: ManualSelector:,
			BackoffLimit:          &backoff,
			ActiveDeadlineSeconds: &deadline,
			Template: k8sv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: labels,
				},
				Spec: k8sv1.PodSpec{
					InitContainers: []k8sv1.Container{}, // Doesn't seem obligatory(?)...
					Containers: []k8sv1.Container{
						{
							Name:  conf.Container.Name,
							Image: conf.Container.Image,
							SecurityContext: &k8sv1.SecurityContext{
								Privileged: &falseVal,
							},
							ImagePullPolicy: pullPolicy,
							Resources: k8sv1.ResourceRequirements{
								Limits: k8sv1.ResourceList{
									k8sv1.ResourceCPU:    resource.MustParse(conf.Container.CPULimit),
									k8sv1.ResourceMemory: resource.MustParse(conf.Container.MemoryLimit),
								},
								Requests: k8sv1.ResourceList{
									k8sv1.ResourceCPU:    resource.MustParse(conf.Container.CPULimit),
									k8sv1.ResourceMemory: resource.MustParse(conf.Container.MemoryLimit),
								},
							},
							Env: []k8sv1.EnvVar{
								{
									Name:  "GEN3_HOSTNAME",
									Value: os.Getenv("GEN3_HOSTNAME"),
								},
								{
									Name:  "INPUT_DATA",
									Value: inputData,
								},
								{
									Name:  "ACCESS_TOKEN",
									Value: accessToken,
								},
								{
									Name:  "DICTIONARY_URL",
									Value: os.Getenv("DICTIONARY_URL"),
								},
								{
									Name:  "BUCKET_NAME",
									Value: pelicanCreds.BucketName,
								},
								{
									Name:  "S3_KEY",
									Value: pelicanCreds.Key,
								},
								{
									Name:  "S3_SECRET",
									Value: pelicanCreds.Secret,
								},
								{
									Name:  "DB_HOST",
									Value: peregrineCreds.DbHost,
								},
								{
									Name:  "DB_USERNAME",
									Value: peregrineCreds.DbUsername,
								},
								{
									Name:  "DB_PASSWORD",
									Value: peregrineCreds.DbPassword,
								},
								{
									Name:  "DB_DATABASE",
									Value: peregrineCreds.DbDatabase,
								},
							},
							VolumeMounts: []k8sv1.VolumeMount{},
						},
					},
					RestartPolicy:    restartPolicy,
					Volumes:          []k8sv1.Volume{},
					ImagePullSecrets: []k8sv1.LocalObjectReference{},
				},
			},
		},
		// Optional, not used by pach: JobStatus:,
	}

	newJob, err := jobsClient.Create(batchJob)
	if err != nil {
		return nil, err
	}
	fmt.Println("New job name: ", newJob.Name)
	ji := JobInfo{}
	ji.Name = newJob.Name
	ji.UID = string(newJob.GetUID())
	ji.Status = jobStatusToString(&newJob.Status)
	return &ji, nil
}

func getPodMatchingJob(jobname string) *k8sv1.Pod {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	pods, err := clientset.CoreV1().Pods(kubectlNamespace).List(metav1.ListOptions{})
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, jobname) {
			return &pod
		}
	}
	return nil
}

func getJobLogs(jobid string) (*JobOutput, error) {
	job, err := getJobByID(getJobClient(), jobid)
	if err != nil {
		return nil, err
	}
	if job.Labels["app"] != "sowerjob" {
		return nil, fmt.Errorf("job with jobid %s not found", jobid)
	}

	pod := getPodMatchingJob(job.Name)
	if pod == nil {
		return nil, fmt.Errorf("Pod not found")
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	podLogOptions := k8sv1.PodLogOptions{}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	podLogs, err := req.Stream()
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, fmt.Errorf("Error copying output")
	}
	str := buf.String()

	ji := JobOutput{}
	ji.Output = str
	return &ji, nil

}

// GetRandString returns a random string of lenght N
func GetRandString(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func jobOlderThan(status *batchv1.JobStatus, cutoffSeconds int32) bool {
	then := time.Now().Add(time.Duration(-cutoffSeconds) * time.Second)
	return status.StartTime.Time.Before(then)
}

func StartMonitoringProcess() {
	jc := getJobClient()
	deleteOption := metav1.NewDeleteOptions(120)
	var deletionPropagation metav1.DeletionPropagation = "Background"
	deleteOption.PropagationPolicy = &deletionPropagation
	for {
		jobsList, err := jc.List(metav1.ListOptions{LabelSelector: "app=sowerjob"})

		if err != nil {
			fmt.Println("Monitoring error: ", err)
			time.Sleep(30 * time.Second)
			continue
		}

		for _, job := range jobsList.Items {
			k8sJob, err := getJobStatusByID(string(job.GetUID()))
			if err != nil {
				fmt.Println("Can't get job status by UID: ", job.Name, err)
			} else {
				if k8sJob.Status == "Unknown" || k8sJob.Status == "Running" {
					continue
				} else {
					if jobOlderThan(&job.Status, 1800) {
						fmt.Println("Deleting old job: ", job.Name)
						if err = jc.Delete(job.Name, deleteOption); err != nil {
							fmt.Println("Error deleting job : ", job.Name, err)
						}
					}
				}
			}

		}

		time.Sleep(30 * time.Second)
	}
}
