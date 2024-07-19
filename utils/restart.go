package utils

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

func Restart(ctx context.Context, client *KubeClient, podContains string, wait bool) (int, error) {

	podsGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	podsGVR, err := client.ResourceFor(podsGVR)
	if err != nil {
		return 0, err
	}

	podsList, err := client.Resource(podsGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	pods := make(map[types.UID]metav1.Object)
	for _, p := range podsList.Items {
		if strings.Contains(strings.ToLower(p.GetName()), podContains) {
			pods[p.GetUID()] = &p
		}
	}

	if len(pods) == 0 {
		return 0, nil
	}

	deletedSignal := make(chan interface{}, len(pods))
	if wait {
		factory := dynamicinformer.NewDynamicSharedInformerFactory(client, time.Minute)
		podsInformer := factory.ForResource(podsGVR).Informer()
		go podsInformer.Run(ctx.Done())

		podsInformer.AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch obj := obj.(type) {
				case metav1.Object:
					_, ok := pods[obj.GetUID()]
					return ok
				case cache.DeletedFinalStateUnknown:
					switch obj := obj.Obj.(type) {
					case metav1.Object:
						_, ok := pods[obj.GetUID()]
						return ok
					}
				}
				return false
			},
			Handler: cache.ResourceEventHandlerFuncs{
				DeleteFunc: func(obj interface{}) {
					deletedSignal <- new(any)
				},
			},
		})
	}

	fieldManager := uuid.NewString()

	var restartCount int
	for _, p := range pods {
		if err := restartPodController(ctx, client, p, fieldManager); err != nil {
			return restartCount, err
		}
		restartCount++
	}

	if wait {
		for range len(pods) {
			<-deletedSignal
		}
	}

	return len(pods), nil
}
