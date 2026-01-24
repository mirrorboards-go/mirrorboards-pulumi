package stacks

import (
	"encoding/base64"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GenerateDockerPullImageConfigJSON generates a docker config JSON for image pull secrets.
func GenerateDockerPullImageConfigJSON(registryURL string, username pulumi.StringInput, token pulumi.StringInput) pulumi.StringMap {
	dockerConfigJSON := pulumi.Sprintf(`{
		"auths": {
			"%s": {
				"username": "%s",
				"password": "%s",
				"auth": "%s"
			}
		}
	}`, registryURL, username, token,
		pulumi.All(username, token).ApplyT(func(args []interface{}) string {
			user := args[0].(string)
			pass := args[1].(string)
			return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		}),
	)

	return pulumi.StringMap{
		".dockerconfigjson": dockerConfigJSON,
	}
}
