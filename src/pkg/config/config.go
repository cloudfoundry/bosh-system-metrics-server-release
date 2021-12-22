package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	// TODO: required / default values?
	EgressPort  int    `yaml:"egress-port"`
	IngressPort int    `yaml:"ingress-port"`
	CertPath    string `yaml:"metrics-cert"`
	KeyPath     string `yaml:"metrics-key"`

	UaaURL            string `yaml:"uaa-url"`
	UaaCA             string `yaml:"uaa-ca"`
	UaaClientIdentity string `yaml:"uaa-client-identity"`
	UaaClientPassword string `yaml:"uaa-client-password"`

	HealthPort int `yaml:"health-port"`
	PProfPort  int `yaml:"pprof-port"`
}

func Read(configFilePath string) (Config, error) {
	configContents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return Config{}, err
	}

	c := Config{}
	err = yaml.Unmarshal(configContents, &c)
	return c, err
}
