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
	Name          string    `json:"name"`
	Container     Container `json:"container"`
	RestartPolicy string    `json:"restart_policy"`
}

func loadConfig(config string) SowerConfig {
	plan, _ := ioutil.ReadFile(config)
	var data SowerConfig
	_ = json.Unmarshal(plan, &data)
	return data
}

type PelicanCreds struct {
	BucketName string `json:"manifest_bucket_name"`
	Hostname   string `json:"hostname"`
	Key        string `json:"aws_access_key_id"`
	Secret     string `json:"aws_secret_access_key"`
}

func loadPelicanCreds(file string) PelicanCreds {
	credsFile, _ := ioutil.ReadFile(file)
	var pelicanCreds PelicanCreds
	json.Unmarshal(credsFile, &pelicanCreds)
	return pelicanCreds
}

type PeregrineCreds struct {
	FenceHost       string `json:"fence_host"`
	FenceUsername   string `json:"fence_username"`
	FencePassword   string `json:"fence_password"`
	FenceDatabase   string `json:"fence_database"`
	DbHost          string `json:"db_host"`
	DbUsername      string `json:"db_username"`
	DbPassword      string `json:"db_password"`
	DbDatabase      string `json:"db_database"`
	GdcAPISecretKey string `json:"gdcapi_secret_key"`
	Hostname        string `json:"hostname"`
}

func loadPeregrineCreds(file string) PeregrineCreds {
	peregrineFile, _ := ioutil.ReadFile(file)
	var peregrineCreds PeregrineCreds
	json.Unmarshal(peregrineFile, &peregrineCreds)
	return peregrineCreds
}
