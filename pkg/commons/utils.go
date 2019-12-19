package commons

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusKey = "mutatingflow.nais.io/status"
)

type Parameters struct {
	CertFile       string
	KeyFile        string
	LogFormat      string
	LogLevel       string
	Teams		   string
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
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
func MutationRequired(metadata metav1.ObjectMeta, annotation string) bool {
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		return true
	}

	_, ok := annotations[annotation]
	if !ok {
		return false
	}

	status := annotations[StatusKey]
	if strings.ToLower(status) == "injected" {
		return false
	}

	return false
}
