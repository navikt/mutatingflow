package commons

import (
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusKey = "mutatingflow.nais.io/status"
)

var (
	proxyEnvs = map[string]string{
		"NO_PROXY":    "localhost,127.0.0.1,10.254.0.1,.local,.adeo.no,.nav.no,.aetat.no,.devillo.no,.oera.no,.nais.io",
		"HTTP_PROXY":  "http://webproxy.nais:8088",
		"HTTPS_PROXY": "http://webproxy.nais:8088",
	}
)

type Parameters struct {
	CertFile       string
	KeyFile        string
	LogFormat      string
	LogLevel       string
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func createEnvVars(envs map[string]string) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	for key, value := range envs {
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  key,
				Value: value,
			})
	}
	return envVars
}

func GetProxyEnvVars() []corev1.EnvVar {
	return createEnvVars(proxyEnvs)
}

func PatchStatusAnnotation(target map[string]string) PatchOperation {
	if target == nil {
		return PatchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				"mutatingflow.nais.io/status": "injected",
			},
		}
	}
	if target[StatusKey] == "" {
		return PatchOperation{
			Op:    "add",
			Path:  "/metadata/annotations/mutatingflow.nais.io~1status",
			Value: "injected",
		}
	}

	return PatchOperation{
		Op:    "replace",
		Path:  "/metadata/annotations/" + StatusKey,
		Value: "injected",
	}
}

// MutationRequired will only modify pods with the annotation
func MutationRequired(metadata metav1.ObjectMeta, label string) bool {
	annotations := metadata.GetAnnotations()
	labels := metadata.GetLabels()
	if labels == nil {
		return true
	}

	log.Infof("Pod: Label %s", label)
	log.Infof("Pod: Labels %s", labels)

	_, ok := labels[label]
	if !ok {
		return false
	}

	log.Infof("Pod: Labels %s", labels)

	status := annotations[StatusKey]
	if strings.ToLower(status) == "injected" {
		return false
	}

	return true
}
