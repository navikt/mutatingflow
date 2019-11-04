package notebooks

import (
	"encoding/json"
	"github.com/navikt/mutatingflow/pkg/commons"
	"github.com/navikt/mutatingflow/pkg/vault"
	"k8s.io/api/admission/v1beta1"
	"strings"

	"github.com/navikt/mutatingflow/pkg/apis/notebook/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "github.com/sirupsen/logrus"
)

func mutatePodSpec(spec corev1.PodSpec, team string) corev1.PodSpec {
	container := spec.Containers[0]

	spec.InitContainers = append(spec.InitContainers, vault.GetInitContainer(team))
	spec.Volumes = append(spec.Volumes, vault.GetVolume())
	container.VolumeMounts = append(container.VolumeMounts, vault.GetVolumeMount())

	container.Env = append(container.Env, commons.GetProxyEnvVars()...)
	container.Env = append(container.Env, commons.GetDataverkEnvVars()...)

	spec.Containers[0] = container

	spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{Name: "gpr-credentials"},
	}
	return spec
}

func patchPodTemplate(spec corev1.PodSpec, team string) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  "/spec/template/spec",
		Value: mutatePodSpec(spec, team),
	}
}

func createPatch(notebook *v1alpha1.Notebook, team string) ([]byte, error) {
	var patch []commons.PatchOperation
	patch = append(patch, patchPodTemplate(notebook.Spec.Template.Spec, team))
	patch = append(patch, commons.PatchStatusAnnotation(notebook.Annotations))
	return json.Marshal(patch)
}

func mutationRequired(metadata *metav1.ObjectMeta) bool {
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		return true
	}

	status := annotations[commons.StatusKey]
	required := true
	if strings.ToLower(status) == "injected" {
		required = false
	}

	return required
}

func MutateNotebook(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
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

	if !mutationRequired(&notebook.ObjectMeta) {
		log.Infof("Notebook: Skipping mutation for %s/%s due to policy check", notebook.Namespace, notebook.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := createPatch(&notebook, request.Namespace)
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
