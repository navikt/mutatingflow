package pipelines

import (
	"encoding/json"
	"fmt"
	"github.com/navikt/mutatingflow/pkg/commons"
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

func patchPodContainerEnvs(env []corev1.EnvVar, index int) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  fmt.Sprintf("/spec/containers/%d/env", index),
		Value: append(env, commons.AddEnvs(env, commons.Envs)...),
	}
}

func patchPodContainerVolumeMounts(mounts []corev1.VolumeMount, index int) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  fmt.Sprintf("/spec/containers/%d/volumeMounts", index),
		Value: append(mounts, commons.AddCaBundleVolumeMounts(mounts)...),
	}
}

func patchContainer(container corev1.Container, index int) []commons.PatchOperation {
	return []commons.PatchOperation{
		patchPodContainerEnvs(container.Env, index),
		patchPodContainerVolumeMounts(container.VolumeMounts, index),
	}
}

func patchVolumes(spec corev1.PodSpec) commons.PatchOperation {
	return commons.PatchOperation{
		Op: "add",
		Path: "/spec/volumes",
		Value: append(spec.Volumes, commons.GetCaBundleVolumes()),
	}
}

func createPatch(pod *corev1.Pod) ([]byte, error) {
	var patch []commons.PatchOperation

	patch = append(patch, patchContainer(pod.Spec.Containers[0], 0)...)
	patch = append(patch, patchContainer(pod.Spec.Containers[1], 1)...)
	patch = append(patch, commons.PatchStatusAnnotation(pod.Annotations))
	patch = append(patch, patchVolumes(pod.Spec))
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

func MutatePod(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
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

	patchBytes, err := createPatch(&pod)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("Pod: Mutated")
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
