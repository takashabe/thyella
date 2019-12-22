package main

import (
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/takashabe/thyella/thyella"
)

type Env struct {
	ProjectID string   `envconfig:"project_id"`
	Cluter    string   `envconfig:"cluster"`
	NodePools []string `envconfig:"node_pools"`
}

func main() {
	var e Env
	if err := envconfig.Process("thyella", &e); err != nil {
		log.Fatal(err)
	}

	kaasClient, err := thyella.NewGKEClient(e.ProjectID)
	if err != nil {
		log.Fatal(err)
	}
	k8sClient, err := thyella.NewK8sClient()
	if err != nil {
		log.Fatal(err)
	}

	p := thyella.Thyella{
		KaasClient: kaasClient,
		K8sClient:  k8sClient,
	}
	if err := p.Run(e.Cluter, e.NodePools); err != nil {
		log.Fatal(err)
	}
}
