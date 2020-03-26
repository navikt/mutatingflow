package pipelines

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/navikt/mutatingflow/pkg/commons"
	"github.com/navikt/mutatingflow/pkg/vault"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// notebookNameLabel is the label we use to check if a pod is of the notebook type
	notebookNameLabel = "notebook-name"
)

func mutateContainer(container corev1.Container) corev1.Container {
	container.VolumeMounts = append(container.VolumeMounts, vault.GetVolumeMount())
	return container
}

func patchContainer(container corev1.Container, index int) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "replace",
		Path:  fmt.Sprintf("/spec/containers/%d", index),
		Value: mutateContainer(container),
	}
}

func getContainerByName(containers []corev1.Container, name string) (corev1.Container, int, error) {
	for i:=0; i<len(containers); i++ {
		if containers[i].Name == name {
			return containers[i], i, nil
		}
	}
	return corev1.Container{}, -1, errors.New("No container with name" + name)
}

func patchVaultInitContainer(team string) ([]commons.PatchOperation, error) {
	return []commons.PatchOperation{
		{
			Op:    "add",
			Path:  "/spec/volumes/-",
			Value: vault.GetVolume(),
		},
		{
			Op:    "add",
			Path:  "/spec/initContainers",
			Value: []corev1.Container{vault.GetInitContainer(team)},
		},
	}, nil
}

func patchImagePullSecrets() commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  "/spec/imagePullSecrets",
		Value: []corev1.LocalObjectReference{{Name: "gpr-credentials"}},
	}
}

func createPatch(pod *corev1.Pod, team string) ([]byte, error) {
	var patch []commons.PatchOperation
	vaultPatches, err := patchVaultInitContainer(team)
	if err != nil {
		return nil, err
	}
	patch = append(patch, vaultPatches...)

	mainContainer, index, err := getContainerByName(pod.Spec.Containers, "main")
	if err != nil {
		return nil, err
	}
	patch = append(patch, patchContainer(mainContainer, index))

	patch = append(patch, patchImagePullSecrets())
	patch = append(patch, commons.PatchStatusAnnotation(pod.Annotations))
	return json.Marshal(patch)
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

	if !commons.MutationRequired(pod.ObjectMeta, notebookNameLabel) {
		log.Infof("Pod: Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := createPatch(&pod, request.Namespace)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Info("Pod: Mutated")
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
