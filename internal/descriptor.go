package internal

import (
	"context"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type authProviderDescribeStep struct {
	name   string
	config map[string]any
}

func newAuthProviderDescribeStep(name string, config map[string]any) (sdk.StepInstance, error) {
	return &authProviderDescribeStep{name: name, config: config}, nil
}

func (s *authProviderDescribeStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, current, _, _ map[string]any) (*sdk.StepResult, error) {
	values := mergeMaps(s.config, current)
	providerID := firstNonEmpty(values, "provider_id", "providerId")
	if providerID == "" {
		providerID = "scalekit"
	}
	environmentURL := firstNonEmpty(values, "environment_url", "environmentUrl", "base_url", "baseUrl")
	return &sdk.StepResult{Output: map[string]any{
		"providers": []map[string]any{scalekitProviderDescriptor(providerID, environmentURL)},
	}}, nil
}

func scalekitProviderDescriptor(providerID, environmentURL string) map[string]any {
	return map[string]any{
		"id":             providerID,
		"label":          "Scalekit",
		"description":    "Scalekit enterprise SSO and SCIM administration integration.",
		"categories":     []string{"enterprise_sso", "directory_sync"},
		"implementation": "workflow-plugin-scalekit",
		"version":        Version,
		"docs_url":       "https://docs.scalekit.com",
		"support_level":  "management",
		"capabilities": []map[string]any{
			scalekitCapability("scalekit_connections", "SSO connections", "enterprise_sso", "Create, read, list, enable, disable, and delete SSO connections through the official Scalekit Go SDK.", []string{"scalekit.connections.read", "scalekit.connections.write"}, scalekitFields(environmentURL)),
			scalekitCapability("scalekit_directories", "SCIM directories", "directory_sync", "Create, read, list, enable, disable, and delete SCIM directory connections through the official Scalekit Go SDK.", []string{"scalekit.directories.read", "scalekit.directories.write"}, scalekitFields(environmentURL)),
			scalekitCapability("scalekit_directory_resources", "Directory users and groups", "directory_sync", "Read synchronized directory users and groups through the official Scalekit Go SDK.", []string{"scalekit.directory.resources.read"}, scalekitFields(environmentURL)),
		},
	}
}

func scalekitCapability(key, label, category, description string, appScopes []string, fields []map[string]any) map[string]any {
	return map[string]any{
		"key":                key,
		"label":              label,
		"category":           category,
		"description":        description,
		"supported":          true,
		"app_scopes":         appScopes,
		"admin_read_scopes":  []string{"admin.auth.providers.read"},
		"admin_write_scopes": []string{"admin.auth.providers.write"},
		"config_fields":      fields,
	}
}

func scalekitFields(environmentURL string) []map[string]any {
	return []map[string]any{
		scalekitField("scalekit_environment_url", "Environment URL", "url", "Scalekit environment URL, for example https://example.scalekit.com.", "Use the environment URL from Scalekit API configuration.", false, true, optionIfSet(strings.TrimRight(environmentURL, "/"))),
		scalekitField("scalekit_client_id", "Client ID", "text", "Scalekit environment client ID.", "Store this with the matching client secret in Workflow configuration.", false, true, nil),
		scalekitField("scalekit_client_secret", "Client secret", "secret", "Scalekit environment client secret.", "Write-only secret. Rotate regularly and store in a Workflow secret source.", true, true, nil),
	}
}

func scalekitField(key, label, inputType, description, helpText string, secret, required bool, options []map[string]any) map[string]any {
	return map[string]any{
		"key":         key,
		"label":       label,
		"input_type":  inputType,
		"description": description,
		"help_text":   helpText,
		"secret":      secret,
		"required":    required,
		"options":     options,
	}
}

func optionIfSet(value string) []map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []map[string]any{{"value": value, "label": value}}
}
