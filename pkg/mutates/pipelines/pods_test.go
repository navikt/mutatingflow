package pipelines

import (
	"fmt"
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
			},
		},
	}
	bytes, err := createPatch(&pod)
	assert.NoError(t, err)
	fmt.Print(string(bytes))
	assert.True(t, false)
}
