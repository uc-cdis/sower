package handlers

import (
	"encoding/json"
	"io/ioutil"
)

type Container struct {
	Name       string
	Image      string
	PullPolicy string
	Env        []string
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
