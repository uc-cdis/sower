package handlers

import (
    "fmt"
    "k8s.io/client-go/kubernetes"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    k8sv1 "k8s.io/api/core/v1"
    batchv1 "k8s.io/api/batch/v1"
    "k8s.io/client-go/rest"
)

var (
	trueVal    = true
	falseVal   = false
)

func createK8sJob() {
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
	//piJob, err := jobsClient.Get("pi")
	//fmt.Printf("piJob Name: %v\n", piJob.Name)

	jobsList, err := jobsClient.List(metav1.ListOptions{})

	// Loop over all jobs and print their name
	for i, job := range jobsList.Items {
		fmt.Printf("Job %d: %s\n", i, job.Name)
	}

	// For an example of how to create jobs, see this file:
	// https://github.com/pachyderm/pachyderm/blob/805e63/src/server/pps/server/api_server.go#L2320-L2345
	batchJob := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "k8sexp-testjob",
			Labels: make(map[string]string),
		},
		Spec: batchv1.JobSpec{
			// Optional: Parallelism:,
			// Optional: Completions:,
			// Optional: ActiveDeadlineSeconds:,
			// Optional: Selector:,
			// Optional: ManualSelector:,
			Template: k8sv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "k8sexp-testpod",
					Labels: make(map[string]string),
				},
				Spec: k8sv1.PodSpec{
					InitContainers: []k8sv1.Container{}, // Doesn't seem obligatory(?)...
					Containers: []k8sv1.Container{
						{
							Name:    "k8sexp-testimg",
							Image:   "perl",
							Command: []string{"sleep", "10"},
							SecurityContext: &k8sv1.SecurityContext{
								Privileged: &falseVal,
							},
							ImagePullPolicy: k8sv1.PullPolicy(k8sv1.PullIfNotPresent),
							Env:             []k8sv1.EnvVar{},
							VolumeMounts:    []k8sv1.VolumeMount{},
						},
					},
					RestartPolicy:    "Never",
					Volumes:          []k8sv1.Volume{},
					ImagePullSecrets: []k8sv1.LocalObjectReference{},
				},
			},
		},
		// Optional, not used by pach: JobStatus:,
	}

	newJob, err := jobsClient.Create(batchJob)

	fmt.Println("New job name: ", newJob.Name)
}
