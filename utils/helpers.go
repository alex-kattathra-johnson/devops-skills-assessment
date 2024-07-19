package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var registeredControllers = make(map[types.UID]interface{})

func restartPodController(ctx context.Context, client *KubeClient, pod metav1.Object, fieldManager string) error {
	var ownerRef *metav1.OwnerReference

	var skip bool
	for _, o := range pod.GetOwnerReferences() {
		if o.Controller == nil || !*o.Controller {
			continue
		}
		if _, ok := registeredControllers[o.UID]; ok {
			skip = true
			break
		}
		registeredControllers[o.UID] = new(any)
		ownerRef = &o
		break
	}

	if skip {
		return nil
	}

	if ownerRef != nil {
		controllerString := strings.ToLower(fmt.Sprintf("%s/%s/%s/%s", ownerRef.APIVersion, ownerRef.Kind, pod.GetNamespace(), ownerRef.Name))

		gvr := schema.GroupVersionResource{
			Group:    strings.ToLower(strings.Split(ownerRef.APIVersion, "/")[0]),
			Version:  strings.ToLower(strings.Split(ownerRef.APIVersion, "/")[1]),
			Resource: strings.ToLower(ownerRef.Kind),
		}

		gvr, err := client.ResourceFor(gvr)
		if err != nil {
			return fmt.Errorf("could not parse GroupVersionResource for %s: %s", controllerString, err)
		}

		data, err := json.Marshal(map[string]interface{}{
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]string{
							"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
						},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("could not marshal patch for %s: %s", controllerString, err)
		}

		ctrlClient := client.Resource(gvr).Namespace(pod.GetNamespace())
		_, err = ctrlClient.Patch(ctx, ownerRef.Name, types.StrategicMergePatchType, data, metav1.PatchOptions{FieldManager: fieldManager})
		if err != nil {
			return fmt.Errorf("could not patch %s: %s", controllerString, err)
		}
		log.Debugf("restarted %s", controllerString)
		return nil
	}
	return fmt.Errorf("the pod %s/%s does not have an associated controller", pod.GetNamespace(), pod.GetName())
}
