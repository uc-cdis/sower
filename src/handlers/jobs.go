package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchtypev1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
)

var (
	trueVal  = true
	falseVal = false
)

type JobsArray struct {
	JobInfo []JobInfo `json:"jobs"`
}

type JobInfo struct {
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type JobOutput struct {
	Output   string `json:"output"`
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
	jobsClient := batchClient.Jobs("default")
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

	// https://kubernetes.io/docs/api-reference/batch/v1/definitions/#_v1_jobstatus
	if status.Succeeded >= 1 {
		return "Completed"
	}
	if status.Failed >= 1 {
		return "Failed"
	}
	if status.Active >= 1 {
		return "Running"
	}
	return "Unknown"
}

func createK8sJob(inputData string, accessToken string) (*JobInfo, error) {
	jobsClient := getJobClient()
	randname, _ := GetRandString(5)
	name := fmt.Sprintf("simu-%s", randname)
	fmt.Println("input data: ", inputData)
	var deadline int64 = 600
	labels := make(map[string]string)
	labels["app"] = "sowerjob"
	// For an example of how to create jobs, see this file:
	// https://github.com/pachyderm/pachyderm/blob/805e63/src/server/pps/server/api_server.go#L2320-L2345
	batchJob := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: batchv1.JobSpec{
			// Optional: Parallelism:,
			// Optional: Completions:,
			// Optional: ActiveDeadlineSeconds:,
			// Optional: Selector:,
			// Optional: ManualSelector:,
			ActiveDeadlineSeconds: &deadline,
			Template: k8sv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: make(map[string]string),
				},
				Spec: k8sv1.PodSpec{
					InitContainers: []k8sv1.Container{}, // Doesn't seem obligatory(?)...
					Containers: []k8sv1.Container{
						{
							Name:  "job-task",
							Image: "quay.io/cdis/simu_demo:latest",
							SecurityContext: &k8sv1.SecurityContext{
								Privileged: &falseVal,
							},
							ImagePullPolicy: k8sv1.PullPolicy(k8sv1.PullIfNotPresent),
							Env: []k8sv1.EnvVar{
								{
									Name:  "INPUT_DATA",
									Value: inputData,
								},
								{
									Name:  "ACCESS_TOKEN",
									Value: accessToken,
								},
							},
							VolumeMounts: []k8sv1.VolumeMount{},
						},
					},
					RestartPolicy:    k8sv1.RestartPolicyNever,
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
	pods, err := clientset.CoreV1().Pods("default").List(metav1.ListOptions{})
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
	//if job.Labels["app"] != "sowerjob" {
	//	return nil, fmt.Errorf("job with jobid %s not found", jobid)
	//}

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
        return nil, fmt.Errorf("Error opening stream")
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
func GetRandString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(base64.RawURLEncoding.EncodeToString(b)), nil
}
