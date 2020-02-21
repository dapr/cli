package kubernetes

import (
	"fmt"
	"os"
	"testing"

	"github.com/magiconair/properties/assert"
)

func KubernetesClientConfig(t *testing.T) {

	t.Run("Initialize client when the only path is configured", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "/home/user/.kube/config")
		_, err := Client()
		assert.Equal(t, err, nil)
	})

	t.Run("Initialize client when only one path is configured and the only one delimiter found belongs to the os path", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "C:\\Users\\DummyUser\\.kube\\config")
		_, err := Client()
		assert.Equal(t, err, nil)
	})

	t.Run("Return error when multiple kubeconfigs are configured", func(t *testing.T) {

		config := "C:\\Users\\DummyUser\\.kube\\config:C:\\Config\\.kube\\config"
		os.Setenv("KUBECONFIG", config)
		_, err := Client()

		assert.Equal(t, err.Error(), fmt.Sprintf("multiple kubeconfigs in KUBECONFIG environment variable - %s", config))

		config = "/home/dummy/.kube/config:/home/dummy/k8s/.kube/config"
		os.Setenv("KUBECONFIG", config)
		_, err = Client()

		assert.Equal(t, err.Error(), fmt.Sprintf("multiple kubeconfigs in KUBECONFIG environment variable - %s", config))
	})
}
