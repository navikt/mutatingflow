package pipelines

import (
	"encoding/json"
	"fmt"
	"github.com/navikt/mutatingflow/pkg/commons"
	"github.com/navikt/mutatingflow/pkg/vault"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// WorkflowArgoAnnotation is the annotation we use to check if a pod is of the workflow-pipeline type
	WorkflowArgoAnnotation = "workflows.argoproj.io/node-name"
)

func mutateContainer(container corev1.Container) corev1.Container {
	container.VolumeMounts = append(container.VolumeMounts, commons.GetCaBundleVolumeMounts()...)
	container.VolumeMounts = append(container.VolumeMounts, vault.GetVolumeMount())
	container.Env = append(container.Env, commons.GetDataverkEnvVars()...)
	container.Env = append(container.Env, commons.GetProxyEnvVars()...)
	return container
}

func patchContainer(container corev1.Container, index int) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "replace",
		Path:  fmt.Sprintf("/spec/containers/%d", index),
		Value: mutateContainer(container),
	}
}

func patchCaBundleVolumes(volumes []corev1.Volume) commons.PatchOperation {
	path := "/spec/volumes"
	if len(volumes) > 0 {
		path += "/-"
	}
	return commons.PatchOperation{
		Op:    "add",
		Path:  path,
		Value: commons.GetCaBundleVolume(),
	}
}

func patchVaultInitContainer(parameters commons.Parameters) []commons.PatchOperation {
	return []commons.PatchOperation{
		{
			Op:    "add",
			Path:  "/spec/volumes/-",
			Value: vault.GetVolume(),
		},
		{
			Op:    "add",
			Path:  "/spec/initContainers",
			Value: []corev1.Container{vault.GetInitContainer(parameters)},
		},
	}
}
func createPatch(pod *corev1.Pod, parameters commons.Parameters) ([]byte, error) {
	var patch []commons.PatchOperation
	patch = append(patch, patchVaultInitContainer(parameters)...)
	patch = append(patch, patchCaBundleVolumes(pod.Spec.Volumes))
	patch = append(patch, patchContainer(pod.Spec.Containers[0], 0))
	patch = append(patch, patchContainer(pod.Spec.Containers[1], 1))
	patch = append(patch, commons.PatchStatusAnnotation(pod.Annotations))
	return json.Marshal(patch)
}

// mutationRequired will only modify pods with the "workflows.argoproj.io/node-name" annotation
func mutationRequired(metadata *metav1.ObjectMeta) bool {
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, ok := annotations[WorkflowArgoAnnotation]
	if !ok {
		return false
	}

	status := annotations[commons.StatusKey]
	if strings.ToLower(status) == "injected" {
		return false
	}

	return true
}

func MutatePod(request *v1beta1.AdmissionRequest, parameters commons.Parameters) *v1beta1.AdmissionResponse {
	var pod corev1.Pod

	err := json.Unmarshal(request.Object.Raw, &pod)
	if err != nil {
		log.Errorf("Pod: Couldn't unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("Pod: Namespace=%v Name=%v (%v) patchOperation=%v", request.Namespace, request.Name, pod.Name, request.Operation)

	if !mutationRequired(&pod.ObjectMeta) {
		log.Infof("Pod: Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := createPatch(&pod, parameters)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("Pod: Mutated")
	log.Info(string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
