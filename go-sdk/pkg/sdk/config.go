package sdk

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	ConfigurationServiceURL string `envconfig:"CONFIGURATION_SERVICE" default:"configuration-service:8080"`
	EventBrokerURL          string `envconfig:"EVENTBROKER" default:"nats://keptn-nats"`
	PubSubTopic             string `envconfig:"PUBSUB_TOPIC" default:""`
	HealthEndpointPort      string `envconfig:"HEALTH_ENDPOINT_PORT" default:"8080"`
	HealthEndpointEnabled   bool   `envconfig:"HEALTH_ENDPOINT_ENABLED" default:"true"`
	Location                string `envconfig:"LOCATION" default:"control-plane"`
	K8sDeploymentVersion    string `envconfig:"K8S_DEPLOYMENT_VERSION" default:""`
	K8sDeploymentName       string `envconfig:"K8S_DEPLOYMENT_NAME" default:""`
	K8sNamespace            string `envconfig:"K8S_NAMESPACE" default:""`
	K8sPodName              string `envconfig:"K8S_POD_NAME" default:""`
	K8sNodeName             string `envconfig:"K8S_NODE_NAME" default:""`
}

func newEnvConfig() envConfig {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("failed to process env var: %s", err)
	}
	return env
}
