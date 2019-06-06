package main

import (
	"encoding/json"
	"fmt"
	"github.com/navikt/mutatingflow/pkg/apis/notebook/v1alpha1"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
	// The following list was copied from https://golang.org/src/crypto/x509/root_linux.go.
	// CA injection should work out of the box. Implementations differ across systems, so
	// by mounting the certificates in these places, we increase the chances of successful auto-configuration.
	certFiles = []string{
		"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
		"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
		"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
		"/etc/pki/tls/cacert.pem",                           // OpenELEC
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	}
	envs = map[string]string{
		"REQUESTS_CA_BUNDLE":          "/etc/pki/tls/certs/ca-bundle.crt",
		"DATAVERK_SECRETS_FROM_FILES": "True",
		"DATAVERK_BUCKET_ENDPOINT":    "https://dataverk-s3-api.nais.preprod.local",
		"NO_PROXY":                    "localhost,127.0.0.1,10.254.0.1,.local,.adeo.no,.nav.no,.aetat.no,.devillo.no,.oera.no,.nais.io",
		"HTTP_PROXY":                  "http://webproxy.nais:8088",
		"HTTPS_PROXY":                 "http://webproxy.nais:8088",
	}
)

const (
	statusKey = "mutatingflow.nais.io/status"
)

type WebhookServer struct {
	server     *http.Server
	parameters Parameters
}

type Parameters struct {
	certFile       string
	keyFile        string
	LogFormat      string
	LogLevel       string
	ServiceAccount string
	VaultKvPath    string
	VaultAuthPath  string
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func hasVolume(name string, volumes []corev1.Volume) bool {
	for _, volume := range volumes {
		if volume.Name == name {
			return true
		}
	}
	return false
}

func addSpecVolumes(volumes []corev1.Volume) []corev1.Volume {
	var missingVolumes []corev1.Volume

	if !hasVolume("vault-secrets", volumes) {
		missingVolumes = append(missingVolumes,
			corev1.Volume{
				Name: "vault-secrets",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium: corev1.StorageMediumMemory,
					},
				},
			})
	}
	if !hasVolume("ca-bundle", volumes) {
		missingVolumes = append(missingVolumes,
			corev1.Volume{
				Name: "ca-bundle",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "ca-bundle",
						},
					},
				},
			})
	}
	return missingVolumes
}

func addVaultContainer(initContainers []corev1.Container, parameters Parameters) []corev1.Container {
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

func hasEnv(name string, env []corev1.EnvVar) bool {
	for _, env := range env {
		if env.Name == name {
			return true
		}
	}

	return false
}

func addEnv(env []corev1.EnvVar) []corev1.EnvVar {
	var missingEnv []corev1.EnvVar

	for key, value := range envs {
		if !hasEnv(key, env) {
			missingEnv = append(missingEnv,
				corev1.EnvVar{
					Name:  key,
					Value: value,
				})
		}
	}
	return []corev1.EnvVar{}
}

func hasVolumeMount(name string, volumeMounts []corev1.VolumeMount) bool {
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == name {
			return true
		}
	}
	return false
}

func addContainerVolumeMounts(volumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
	var missingVolumeMounts []corev1.VolumeMount

	if !hasVolumeMount("vault-secrets", volumeMounts) {
		missingVolumeMounts = append(missingVolumeMounts,
			corev1.VolumeMount{
				Name:
				"vault-secrets",
				MountPath: "/var/run/secrets/nais.io/vault",
			})
	}

	if !hasVolumeMount("ca-bundle", volumeMounts) {
		for _, path := range certFiles {
			missingVolumeMounts = append(missingVolumeMounts, corev1.VolumeMount{
				Name:      "ca-bundle",
				MountPath: path,
				SubPath:   "ca-bundle.pem",
			})
		}

		missingVolumeMounts = append(missingVolumeMounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: "/etc/ssl/certs/java/cacerts",
			SubPath:   "ca-bundle.jks",
		})
	}
	return missingVolumeMounts
}

func mutatePodSpec(spec corev1.PodSpec, parameters Parameters) corev1.PodSpec {
	container := spec.Containers[0]
	container.VolumeMounts = append(container.VolumeMounts, addContainerVolumeMounts(container.VolumeMounts)...)
	container.Env = append(container.Env, addEnv(container.Env)...)
	spec.Containers[0] = container
	spec.InitContainers = append(spec.InitContainers, addVaultContainer(spec.InitContainers, parameters)...)
	spec.Volumes = append(spec.Volumes, addSpecVolumes(spec.Volumes)...)
	spec.ServiceAccountName = parameters.ServiceAccount
	return spec
}

func updatePodTemplate(spec corev1.PodSpec, parameters Parameters) patchOperation {
	return patchOperation{
		Op:    "add",
		Path:  "/spec/template/spec",
		Value: mutatePodSpec(spec, parameters),
	}
}

func updateAnnotation(target map[string]string) patchOperation {
	if target == nil || target[statusKey] == "" {
		target = map[string]string{}
		return patchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				statusKey: "injected",
			},
		}
	}

	return patchOperation{
		Op:    "replace",
		Path:  "/metadata/annotations/" + statusKey,
		Value: "injected",
	}
}

func createPatch(notebook *v1alpha1.Notebook, parameters Parameters) ([]byte, error) {
	var patch []patchOperation
	patch = append(patch, updatePodTemplate(notebook.Spec.Template.Spec, parameters))
	patch = append(patch, updateAnnotation(notebook.Annotations))
	return json.Marshal(patch)
}

func mutationRequired(metadata *metav1.ObjectMeta) bool {
	annotations := metadata.GetAnnotations()
	log.Info(annotations)
	if annotations == nil {
		return true
	}

	status := annotations[statusKey]
	log.Info(status)
	required := true
	if strings.ToLower(status) == "injected" {
		required = false
	}

	log.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
	return required
}

func (server *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	request := ar.Request
	var notebook v1alpha1.Notebook
	err := json.Unmarshal(request.Object.Raw, &notebook)
	if err != nil {
		log.Errorf("Couldn't unmarshal raw notebook object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		request.Kind, request.Namespace, request.Name, notebook.Name, request.UID, request.Operation, request.UserInfo)

	if !mutationRequired(&notebook.ObjectMeta) {
		log.Infof("Skipping mutation for %s/%s due to policy check", notebook.Namespace, notebook.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := createPatch(&notebook, server.parameters)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (server *WebhookServer) serve(responseWriter http.ResponseWriter, request *http.Request) {
	var body []byte
	if request.Body != nil {
		if data, err := ioutil.ReadAll(request.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		log.Error("empty body")
		http.Error(responseWriter, "empty body", http.StatusBadRequest)
		return
	}

	contentType := request.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("Content-Type=%s, expected application/json", contentType)
		http.Error(responseWriter, "invalid Content-Type, expected `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = server.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		log.Errorf("Can't encode response: %v", err)
		http.Error(responseWriter, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	if _, err := responseWriter.Write(resp); err != nil {
		log.Errorf("Can't write response: %v", err)
		http.Error(responseWriter, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
