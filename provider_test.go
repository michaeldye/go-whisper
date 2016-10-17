package whisper

import (
    "net/url"
    "testing"
)

func Test_NewConfigure(t *testing.T) {

    url, _ := url.Parse("http://localhost")
    c := NewConfigure("nonce", *url, map[string]string{"hash":"123"}, map[string]string{"sig":"456"}, "deployment", "deploymentSignature", "deploymentUserInfo")

    if c == nil || c.DeploymentUserInfo != "deploymentUserInfo" {
        t.Errorf("Configure object not created correctly %v", c)
    }
}
