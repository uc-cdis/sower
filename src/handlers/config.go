package handlers

import (
	"encoding/json"
	"io/ioutil"
)

// Container Struct to hold the configuration for Job Container
type Container struct {
	Name       string   `json:"name"`
	Image      string   `json:"image"`
	PullPolicy string   `json:"pull_policy"`
	Env        []string `json:"env"`
}

// SowerConfig Struct to hold all the configuration
type SowerConfig struct {
	Container     Container `json:"container"`
	RestartPolicy string    `json:"restart_policy"`
}

func loadConfig(config string) SowerConfig {
	plan, _ := ioutil.ReadFile(config)
	var data SowerConfig
	_ = json.Unmarshal(plan, &data)
	return data
}
