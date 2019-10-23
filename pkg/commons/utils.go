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
	certFiles = []string{
		"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
		"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
		"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
		"/etc/pki/tls/cacert.pem",                           // OpenELEC
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	}
	dataverkEnvs = map[string]string{
		"DATAVERK_SECRETS_FROM_FILES": "True",
		"DATAVERK_BUCKET_ENDPOINT":    "https://dataverk-s3-api.nais.adeo.no",
		"REQUESTS_CA_BUNDLE":          "/etc/pki/tls/certs/ca-bundle.crt",
		"SSL_CERT_FILE":               "/etc/pki/tls/certs/ca-bundle.crt",
		"VKS_SECRET_DEST_PATH":        "/var/run/secrets/nais.io/vault",
	}
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
	ServiceAccount string
	VaultKvPath    string
	VaultAuthPath  string
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

func GetDataverkEnvVars() []corev1.EnvVar {
	return createEnvVars(dataverkEnvs)
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

func GetCaBundleVolumeMounts() []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	for _, path := range certFiles {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: path,
			SubPath:   "ca-bundle.pem",
		})
	}

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "ca-bundle",
		MountPath: "/etc/ssl/certs/java/cacerts",
		SubPath:   "ca-bundle.jks",
	})

	return volumeMounts
}

func GetCaBundleVolume() corev1.Volume {
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
