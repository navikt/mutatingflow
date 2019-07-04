package vault

import (
	"github.com/navikt/mutatingflow/pkg/commons"
	corev1 "k8s.io/api/core/v1"
)

func GetVolume() corev1.Volume {
	return corev1.Volume{
		Name: "vault-secrets",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
}

func GetInitContainer(parameters commons.Parameters) corev1.Container {
	allowPrivilegeEscalation := false
	return corev1.Container{
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
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
	}
}

func GetVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:
		"vault-secrets",
		MountPath: "/var/run/secrets/nais.io/vault",
	}
}
