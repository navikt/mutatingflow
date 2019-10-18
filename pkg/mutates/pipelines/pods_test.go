package pipelines

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestCreatePatch(t *testing.T) {
	t.Run("Create a patch should not return errors", func(t *testing.T) {
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
		_, err := createPatch(&pod, "testTeam")
		assert.NoError(t, err)
	})
}
