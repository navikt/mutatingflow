package pipelines

import (
	"encoding/json"
	"github.com/navikt/mutatingflow/pkg/commons"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// workflowArgoAnnotation is the annotation we use to check if a pod is of the workflow-pipeline type
	workflowArgoAnnotation = "workflows.argoproj.io/node-name"
)

func patchImagePullSecrets() commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  "/spec/imagePullSecrets",
		Value: []corev1.LocalObjectReference{{Name: "gpr-credentials"}},
	}
}

func createPatch(pod *corev1.Pod, team string) ([]byte, error) {
	var patch []commons.PatchOperation
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

	if !commons.MutationRequired(pod.ObjectMeta, workflowArgoAnnotation) {
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
