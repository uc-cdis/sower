package handlers

import (
	"encoding/json"
	"io/ioutil"
)

type Container struct {
	Name       string   `json:"name"`
	Image      string   `json:"image"`
	PullPolicy string   `json:"pull_policy"`
	Env        []string `json:"env"`
}

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
