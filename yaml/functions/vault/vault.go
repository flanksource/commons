package vault

import (
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"gopkg.in/flanksource/yaml.v3"
)

func init() {
	yaml.AddTemplateFunction("vault", vaultFunc)
}

func vaultFunc(args ...string) string {
	var vaultAddr string
	var vaultPath string
	var vaultKey string

	if len(args) < 2 {
		return ""
	} else if len(args) == 2 {
		vaultAddr = os.Getenv("VAULT_ADDR")
		if vaultAddr == "" {
			return ""
		}
		vaultPath = args[0]
		vaultKey = args[1]
	} else {
		vaultAddr = args[0]
		vaultPath = args[1]
		vaultKey = args[2]
	}

	var httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	client, err := api.NewClient(&api.Config{Address: vaultAddr, HttpClient: httpClient})
	if err != nil {
		log.Errorf("Failed to create vault client: %v", err)
		return ""
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		log.Errorf("VAULT_TOKEN is empty")
		return ""
	}
	client.SetToken(token)

	secret, err := client.Logical().Read(vaultPath)
	if err != nil {
		log.Errorf("Failed to get vault path %s: %v", vaultPath, err)
		return ""
	}

	m, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		log.Errorf("Failed to convert data to v2 type data")
		return ""
	}
	value, ok := m[vaultKey]
	if !ok {
		log.Errorf("Failed to get key %s in path %s", vaultKey, vaultPath)
		return ""
	}

	return value.(string)
}
