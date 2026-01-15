package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Interaction struct {
	To       string   `yaml:"to"`
	Actions  []string `yaml:"actions"`
	Port     int      `yaml:"port,omitempty"`
	Protocol string   `yaml:"protocol,omitempty"`
}

type Domain struct {
	AllowedInteractions []Interaction `yaml:"allowed_interactions"`
}

type Policy struct {
	SecurityDomains map[string]Domain `yaml:"security_domains"`
}

var GlobalPolicy Policy

func LoadPolicies() error {
	data, err := os.ReadFile("/policies/inter_domain_interactions.yaml")
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &GlobalPolicy)
}

func IsAllowed(source, target, action string) (bool, string) {
	sourceDomain, exists := GlobalPolicy.SecurityDomains[source]
	if !exists {
		return false, "source domain not found"
	}
	for _, interaction := range sourceDomain.AllowedInteractions {
		if interaction.To == target {
			for _, a := range interaction.Actions {
				if a == action {
					return true, "allowed by policy"
				}
			}
		}
	}
	return false, "action not allowed by policy"
}
