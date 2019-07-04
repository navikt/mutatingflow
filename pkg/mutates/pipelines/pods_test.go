package pipelines

import (
	"github.com/navikt/mutatingflow/pkg/commons"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestCreatePatch(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "Hello-world",
				},
				{
					Name: "Goodbye-world",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "pipeline-runner-token-abcsd12",
					VolumeSource: corev1.VolumeSource{},
				},
			},
		},
	}
	_, err := createPatch(&pod, commons.Parameters{})
	assert.NoError(t, err)
}
