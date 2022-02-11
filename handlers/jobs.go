package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchtypev1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
)

var kubectlNamespace = os.Getenv("POD_NAMESPACE")

// JobInfo is an information about dispatched job
type JobInfo struct {
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// JobOutput to return job output
type JobOutput struct {
	Output string `json:"output"`
}

func (t *JobOutput) JSON() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func getJobClient() batchtypev1.JobInterface {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	// Access jobs. We can't do it all in one line, since we need to receive the
	// errors and manage thgem appropriately
	batchClient := clientset.BatchV1()
	jobClient := batchClient.Jobs(kubectlNamespace)
	return jobClient
}

func getJobByID(jobid string, username string) (*batchv1.Job, error) {
	log.WithField("jobid", jobid).Debug("Get Job By ID")

	jc := getJobClient()

	labelSelector := ""
	if username == "" { // this is needed for StartMonitoringProcess function only
		labelSelector = fmt.Sprintf("app=%s", "sowerjob")
	} else {
		labelSelector = fmt.Sprintf("app=%s,username=%s", "sowerjob", username)
	}

	jobs, err := jc.List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
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

func getJobStatusByID(jobid string, username string) (*JobInfo, error) {
	log.WithField("jobid", jobid).Debug("Get Job Status By ID")

	job, err := getJobByID(jobid, username)
	if err != nil {
		return nil, err
	}
	ji := JobInfo{Name: job.Name, UID: string(job.GetUID()), Status: jobStatusToString(&job.Status)}
	return &ji, nil
}

func listJobs(jc batchtypev1.JobInterface, username string) []JobInfo {
	jobs := []JobInfo{}

	labelSelector := fmt.Sprintf("app=%s,username=%s", "sowerjob", username)

	jobsList, err := jc.List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return jobs
	}

	for _, job := range jobsList.Items {
		ji := JobInfo{Name: job.Name,
			UID: string(job.GetUID()), Status: jobStatusToString(&job.Status)}
		jobs = append(jobs, ji)
	}

	return jobs
}

func jobStatusToString(status *batchv1.JobStatus) string {
	if status == nil {
		return "Unknown"
	}

	// https://pkg.go.dev/k8s.io/api/batch/v1#JobStatus
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

func createK8sJob(currentAction string, inputData string, accessFormat string, accessToken string, userName string, username string) (*JobInfo, error) {
	var availableActions = loadSowerConfigs("/sower_config.json")
	var getCurrentAction = func(s SowerConfig) bool { return s.Action == currentAction }
	var actions = filter(availableActions, getCurrentAction)

	if len(actions) == 0 {
		fmt.Println("ERROR: %V is not in sower config", currentAction)
		return nil, nil
	} else if len(actions) != 1 {
		fmt.Println("ERROR: There is a duplicate in sower config")
		return nil, nil
	}

	var conf = actions[0]

	fmt.Println("config: ", conf)

	jobsClient := getJobClient()
	randname := GetRandString(5)
	name := fmt.Sprintf("%s-%s", conf.Name, randname)
	fmt.Println("input data: ", inputData)
	var deadline int64 = 7200
	var backoff int32 = 1
	labels := make(map[string]string)
	labels["app"] = "sowerjob"
	labels["username"] = username

	annotations := make(map[string]string)
	annotations["gen3username"] = userName

	var privileged = false

	var env = []k8sv1.EnvVar{
		{
			Name:  "INPUT_DATA",
			Value: inputData,
		},
		{
			Name:  "ACCESS_TOKEN",
			Value: accessToken,
		},
		{
			Name:  "ACCESS_FORMAT",
			Value: accessFormat,
		},
	}
	env = append(env, conf.Container.Env...)

	var volumes []k8sv1.Volume
	volumes = append(volumes, conf.Volumes...)

	var volumeMounts []k8sv1.VolumeMount
	volumeMounts = append(volumeMounts, conf.Container.VolumesMounts...)

	var batchJob *batchv1.Job

	// For an example of how to create jobs, see this file:
	// https://github.com/pachyderm/pachyderm/blob/805e63/src/server/pps/server/api_server.go#L2320-L2345
	batchJob = &batchv1.Job{
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
					Containers: []k8sv1.Container{
						{
							Name:  conf.Container.Name,
							Image: conf.Container.Image,
							SecurityContext: &k8sv1.SecurityContext{
								Privileged: &privileged,
							},
							ImagePullPolicy: conf.Container.PullPolicy,
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
							Env:          env,
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy:    conf.RestartPolicy,
					Volumes:          volumes,
					ImagePullSecrets: []k8sv1.LocalObjectReference{},
				},
			},
		},
	}

	if conf.ServiceAccountName != nil {
		var saName string = *conf.ServiceAccountName
		batchJob.Spec.Template.Spec.ServiceAccountName = saName
	}

	if conf.ActiveDeadlineSeconds != nil {
		var deadline int64 = *conf.ActiveDeadlineSeconds
		batchJob.Spec.ActiveDeadlineSeconds = &deadline
	}

	newJob, err := jobsClient.Create(context.TODO(), batchJob, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(err)
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
	pods, err := clientset.CoreV1().Pods(kubectlNamespace).List(context.TODO(), metav1.ListOptions{})
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, jobname) {
			return &pod
		}
	}
	return nil
}

func getJobLogs(jobid string, username string) (*JobOutput, error) {
	job, err := getJobByID(jobid, username)
	if err != nil {
		return nil, err
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
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, fmt.Errorf("Error copying output")
	}
	str := html.UnescapeString(buf.String())
	fmt.Println(str)

	ji := JobOutput{}
	ji.Output = str
	return &ji, nil

}

// GetRandString returns a random string of length N
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
		jobsList, err := jc.List(context.TODO(), metav1.ListOptions{LabelSelector: "app=sowerjob"})

		if err != nil {
			fmt.Println("Monitoring error: ", err)
			time.Sleep(30 * time.Second)
			continue
		}

		for _, job := range jobsList.Items {
			k8sJob, err := getJobStatusByID(string(job.GetUID()), "")
			if err != nil {
				fmt.Println("Can't get job status by UID: ", job.Name, err)
			} else {
				if k8sJob.Status == "Unknown" || k8sJob.Status == "Running" {
					continue
				} else {
					if jobOlderThan(&job.Status, 1800) {
						fmt.Println("Deleting old job: ", job.Name)
						if err = jc.Delete(context.TODO(), job.Name, *deleteOption); err != nil {
							fmt.Println("Error deleting job : ", job.Name, err)
						}
					}
				}
			}

		}

		time.Sleep(30 * time.Second)
	}
}

func deleteJob(UID string, username string) error {
	jc := getJobClient()
	deleteOption := metav1.NewDeleteOptions(120)
	var deletionPropagation metav1.DeletionPropagation = "Background"
	deleteOption.PropagationPolicy = &deletionPropagation
	if job, err := getJobByID(UID, username); err != nil {
		return err
	} else if err = jc.Delete(context.TODO(), job.Name, *deleteOption); err != nil {
		return err
	}
	return nil
}
