package notebooks

import (
	"encoding/json"
	"github.com/navikt/mutatingflow/pkg/commons"
	"k8s.io/api/admission/v1beta1"
	"strings"

	"github.com/navikt/mutatingflow/pkg/apis/notebook/v1alpha1"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	notebookEnvs = map[string]string{
		"DATAVERK_SECRETS_FROM_FILES": "True",
		"DATAVERK_BUCKET_ENDPOINT":    "https://dataverk-s3-api.nais.adeo.no",
	}
)

func addVaultVolume(volumes []corev1.Volume) corev1.Volume {
	if !commons.HasVolume("vault-secrets", volumes) {
		return corev1.Volume{
				Name: "vault-secrets",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium: corev1.StorageMediumMemory,
					},
				},
			}
	}

	return corev1.Volume{}
}

func addVaultContainer(initContainers []corev1.Container, parameters commons.Parameters) []corev1.Container {
	for _, initContainer := range initContainers {
		if initContainer.Name == "vks" {
			return []corev1.Container{}
		}
	}

	return []corev1.Container{
		{
			Image: "navikt/vks:38",
			Name:  "vks",
			Env: []corev1.EnvVar{
				{
					Name:  "VKS_VAULT_ADDR",
					Value: "https://vault.adeo.no",
				},
				{
					Name:  "VKS_AUTH_PATH",
					Value: parameters.VaultAuthPath,
				},
				{
					Name:  "VKS_KV_PATH",
					Value: parameters.VaultKvPath,
				},
				{
					Name:  "VKS_VAULT_ROLE",
					Value: parameters.ServiceAccount,
				},
				{
					Name:  "VKS_SECRET_DEST_PATH",
					Value: "/var/run/secrets/nais.io/vault",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vault-secrets",
					MountPath: "/var/run/secrets/nais.io/vault",
				},
			},
		},
	}
}

func addContainerVolumeMounts(volumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
	var missingVolumeMounts []corev1.VolumeMount

	if !commons.HasVolumeMount("vault-secrets", volumeMounts) {
		missingVolumeMounts = append(missingVolumeMounts,
			corev1.VolumeMount{
				Name:
				"vault-secrets",
				MountPath: "/var/run/secrets/nais.io/vault",
			})
	}

	return missingVolumeMounts
}

func mutatePodSpec(spec corev1.PodSpec, parameters commons.Parameters) corev1.PodSpec {
	container := spec.Containers[0]
	container.VolumeMounts = append(container.VolumeMounts, addContainerVolumeMounts(container.VolumeMounts)...)
	container.VolumeMounts = append(container.VolumeMounts, commons.AddCaBundleVolumeMounts(container.VolumeMounts)...)
	spec.Volumes = append(spec.Volumes, commons.GetCaBundleVolumes())
	container.Env = append(container.Env, commons.AddEnvs(container.Env, notebookEnvs)...)
	container.Env = append(container.Env, commons.AddEnvs(container.Env, commons.Envs)...)
	spec.Containers[0] = container
	spec.InitContainers = append(spec.InitContainers, addVaultContainer(spec.InitContainers, parameters)...)
	spec.Volumes = append(spec.Volumes, addVaultVolume(spec.Volumes))
	spec.ServiceAccountName = parameters.ServiceAccount
	return spec
}

func updatePodTemplate(spec corev1.PodSpec, parameters commons.Parameters) commons.PatchOperation {
	return commons.PatchOperation{
		Op:    "add",
		Path:  "/spec/template/spec",
		Value: mutatePodSpec(spec, parameters),
	}
}

func createPatch(notebook *v1alpha1.Notebook, parameters commons.Parameters) ([]byte, error) {
	var patch []commons.PatchOperation
	patch = append(patch, updatePodTemplate(notebook.Spec.Template.Spec, parameters))
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

func MutateNotebook(request *v1beta1.AdmissionRequest, parameters commons.Parameters) *v1beta1.AdmissionResponse {
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

	patchBytes, err := createPatch(&notebook, parameters)
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
