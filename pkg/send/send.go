package send

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dapr/cli/pkg/api"

	"github.com/dapr/cli/pkg/standalone"
)

func InvokeApp(appID, method, payload string) (string, error) {
	list, err := standalone.List()
	if err != nil {
		return "", err
	}

	for _, lo := range list {
		if lo.AppID == appID {
			r, err := http.Post(fmt.Sprintf("http://localhost:%s/v%s/invoke/%s/method/%s", fmt.Sprintf("%v", lo.HTTPPort), api.RuntimeAPIVersion, lo.AppID, method), "application/json", bytes.NewBuffer([]byte(payload)))
			if err != nil {
				return "", err
			}

			rb, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return "", err
			}

			if len(rb) > 0 {
				return string(rb), nil
			}

			return "", nil
		}
	}

	return "", fmt.Errorf("App ID %s not found", appID)
}
