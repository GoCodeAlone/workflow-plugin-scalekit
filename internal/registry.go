package internal

import (
	"sync"

	scalekit "github.com/scalekit-inc/scalekit-sdk-go/v2"
)

type scalekitSDK interface {
	Connection() scalekit.Connection
	Directory() scalekit.Directory
}

type ScalekitClient struct {
	SDK            scalekitSDK
	EnvironmentURL string
}

var (
	clientMu       sync.RWMutex
	clientRegistry = map[string]*ScalekitClient{}
)

func RegisterClient(name string, client *ScalekitClient) {
	clientMu.Lock()
	defer clientMu.Unlock()
	clientRegistry[name] = client
}

func GetClient(name string) (*ScalekitClient, bool) {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, ok := clientRegistry[name]
	return client, ok
}

func UnregisterClient(name string) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(clientRegistry, name)
}
