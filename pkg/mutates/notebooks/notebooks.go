package notebooks

import (
	"encoding/json"
	"github.com/navikt/mutatingflow/pkg/apis/notebook/v1alpha1"
	"github.com/navikt/mutatingflow/pkg/commons"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// notebookNameAnnotation is the annotation we use to check if a pod is of the notebook type
	notebookNameLabel = "notebook-name"
)

func mutatePodSpec(spec corev1.PodSpec) corev1.PodSpec {
	container := spec.Containers[0]
	container.Env = append(container.Env, commons.GetProxyEnvVars()...)
	spec.Containers[0] = container

	spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{Name: "gpr-credentials"},
	}

	return spec
}

func patchPodTemplate(spec corev1.PodSpec) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  "/spec/template/spec",
		Value: mutatePodSpec(spec),
	}
}

func createPatch(notebook v1alpha1.Notebook) ([]byte, error) {
	var patch []commons.PatchOperation
	patch = append(patch, patchPodTemplate(notebook.Spec.Template.Spec))
	patch = append(patch, commons.PatchStatusAnnotation(notebook.Annotations))
	return json.Marshal(patch)
}

func MutateNotebook(request v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	var notebook v1alpha1.Notebook

	err := json.Unmarshal(request.Object.Raw, &notebook)
	if err != nil {
		log.Errorf("Notebook: Couldn't unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("Notebook: Namespace=%v Name=%v (%v) patchOperation=%v", request.Namespace, request.Name, notebook.Name, request.Operation)

	if !commons.MutationRequired(notebook.ObjectMeta, notebookNameLabel) {
		log.Infof("Notebook: Skipping mutation for %s/%s due to policy check", notebook.Namespace, notebook.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := createPatch(notebook)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Info("Notebook: Mutated")
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
