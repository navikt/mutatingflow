package vault

import (
	"fmt"
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

func GetInitContainer(team string) corev1.Container {
	allowPrivilegeEscalation := false
	return corev1.Container{
		Image: "navikt/vks:46",
		Name:  "vks",
		Env: []corev1.EnvVar{
			{
				Name:  "VKS_VAULT_ADDR",
				Value: "https://vault.adeo.no",
			},
			{
				Name:  "VKS_AUTH_PATH",
				Value: "/kubernetes/prod/kubeflow",
			},
			{
				Name:  "VKS_KV_PATH",
				Value: fmt.Sprintf("/kv/prod/kubeflow/%[1]s/%[1]s", team),
			},
			{
				Name:  "VKS_VAULT_ROLE",
				Value: team,
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
