package commons

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	StatusKey = "mutatingflow.nais.io/status"
)

var (
	// The following list was copied from https://golang.org/src/crypto/x509/root_linux.go.
	// CA injection should work out of the box. Implementations differ across systems, so
	// by mounting the certificates in these places, we increase the chances of successful auto-configuration.
	CertFiles = []string{
		"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
		"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
		"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
		"/etc/pki/tls/cacert.pem",                           // OpenELEC
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	}
	Envs = map[string]string{
		"REQUESTS_CA_BUNDLE": "/etc/pki/tls/certs/ca-bundle.crt",
		"NO_PROXY":           "localhost,127.0.0.1,10.254.0.1,.local,.adeo.no,.nav.no,.aetat.no,.devillo.no,.oera.no,.nais.io",
		"HTTP_PROXY":         "http://webproxy.nais:8088",
		"HTTPS_PROXY":        "http://webproxy.nais:8088",
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

func PatchStatusAnnotation(target map[string]string) PatchOperation {
	if target == nil || target[StatusKey] == "" {
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

func HasVolumeMount(name string, volumeMounts []corev1.VolumeMount) bool {
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == name {
			return true
		}
	}
	return false
}

func AddCaBundleVolumeMounts(volumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
	var missingVolumeMounts []corev1.VolumeMount

	if !HasVolumeMount("ca-bundle", volumeMounts) {
		for _, path := range CertFiles {
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

func HasVolume(name string, volumes []corev1.Volume) bool {
	for _, volume := range volumes {
		if volume.Name == name {
			return true
		}
	}
	return false
}

func GetCaBundleVolumes() corev1.Volume {
	return corev1.Volume{
		Name: "ca-bundle",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "ca-bundle",
				},
			},
		},
	}
}
