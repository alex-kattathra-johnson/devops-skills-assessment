package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type KubeClient struct {
	dynamic.Interface
	meta.RESTMapper
}

func CreateClient(config *rest.Config) (client *KubeClient, err error) {
	client = new(KubeClient)

	client.Interface, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create kube client: %s", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create discovery client: %s", err)
	}

	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, err
	}
	client.RESTMapper = restmapper.NewDiscoveryRESTMapper(groupResources)

	return client, nil
}
