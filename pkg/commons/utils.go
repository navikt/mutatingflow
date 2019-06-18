package commons

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	StatusKey = "mutatingflow.nais.io/status"
)

var (
	Envs = map[string]string{
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
	ServiceAccount string
	VaultKvPath    string
	VaultAuthPath  string
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func hasEnv(name string, env []corev1.EnvVar) bool {
	for _, env := range env {
		if env.Name == name {
			return true
		}
	}

	return false
}

func AddEnvs(orgEnvs []corev1.EnvVar, newEnvs map[string]string) []corev1.EnvVar {
	var missingEnvs []corev1.EnvVar

	for key, value := range newEnvs {
		if !hasEnv(key, orgEnvs) {
			missingEnvs = append(missingEnvs,
				corev1.EnvVar{
					Name:  key,
					Value: value,
				})
		}
	}
	return missingEnvs
}

func UpdateAnnotation(target map[string]string) PatchOperation {
	if target == nil || target[StatusKey] == "" {
		target = map[string]string{}
		return PatchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				StatusKey: "injected",
			},
		}
	}

	return PatchOperation{
		Op:    "replace",
		Path:  "/metadata/annotations/" + StatusKey,
		Value: "injected",
	}
}