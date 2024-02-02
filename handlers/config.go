package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	k8sv1 "k8s.io/api/core/v1"
)

// Container Struct to hold the configuration for Job Container
type Container struct {
	Name          string              `json:"name"`
	Image         string              `json:"image"`
	PullPolicy    k8sv1.PullPolicy    `json:"pull_policy"`
	Env           []k8sv1.EnvVar      `json:"env"`
	VolumesMounts []k8sv1.VolumeMount `json:"volumeMounts"`
	CPULimit      string              `json:"cpu-limit"`
	MemoryLimit   string              `json:"memory-limit"`
}

// SowerConfig Struct to hold all the configuration
type SowerConfig struct {
	Name                  string              `json:"name"`
	Action                string              `json:"action"`
	Container             Container           `json:"container"`
	Volumes               []k8sv1.Volume      `json:"volumes"`
	RestartPolicy         k8sv1.RestartPolicy `json:"restart_policy"`
	ServiceAccountName    *string             `json:"serviceAccountName"`
	ActiveDeadlineSeconds *int64              `json:"activeDeadlineSeconds"`
	TTLSecondsAfterFinished *int32              `json:"ttlSecondsAfterFinished"`
}

func loadSowerConfigs(config string) []SowerConfig {
	plan, _ := ioutil.ReadFile(config)
	var data []SowerConfig
	err := json.Unmarshal(plan, &data)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return nil
	}
	return data
}
