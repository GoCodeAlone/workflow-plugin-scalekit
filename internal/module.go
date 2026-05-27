package internal

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	scalekit "github.com/scalekit-inc/scalekit-sdk-go/v2"
)

type scalekitModule struct {
	name   string
	config map[string]any
}

func newScalekitModule(name string, config map[string]any) (*scalekitModule, error) {
	return &scalekitModule{name: name, config: config}, nil
}

func (m *scalekitModule) Init() error {
	environmentURL := firstNonEmpty(m.config, "environment_url", "environmentUrl", "base_url", "baseUrl", "url")
	if environmentURL == "" {
		return fmt.Errorf("scalekit.provider %q: environment_url is required", m.name)
	}
	if _, err := url.ParseRequestURI(environmentURL); err != nil {
		return fmt.Errorf("scalekit.provider %q: invalid environment_url: %w", m.name, err)
	}
	clientID := firstNonEmpty(m.config, "client_id", "clientId")
	if clientID == "" {
		return fmt.Errorf("scalekit.provider %q: client_id is required", m.name)
	}
	clientSecret := firstNonEmpty(m.config, "client_secret", "clientSecret")
	client := scalekit.NewScalekitClient(environmentURL, clientID, clientSecret)
	RegisterClient(m.name, &ScalekitClient{SDK: client, EnvironmentURL: strings.TrimRight(environmentURL, "/")})
	return nil
}

func (m *scalekitModule) Start(context.Context) error { return nil }

func (m *scalekitModule) Stop(context.Context) error {
	UnregisterClient(m.name)
	return nil
}
