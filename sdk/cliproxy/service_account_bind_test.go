package cliproxy

import (
	"testing"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

func TestServiceLookupBoundAuthIndexIgnoresBindingsWithoutAccountBindStrategy(t *testing.T) {
	const clientKey = "sk-client"

	service := &Service{
		cfg:         &config.Config{},
		coreManager: coreauth.NewManager(nil, nil, nil),
	}
	service.rebuildBindingMap(&config.Config{
		SDKConfig: config.SDKConfig{
			APIKeys:            config.FlexAPIKeyList{clientKey},
			APIKeyAuthBindings: map[string]string{clientKey: "idx-bound"},
		},
		Routing: internalconfig.RoutingConfig{DefaultModelAccount: "idx-default"},
	})

	if got, ok := service.LookupBoundAuthIndex(clientKey); ok || got != "" {
		t.Fatalf("binding must be inactive outside account-bind: got=%q ok=%v", got, ok)
	}
}

func TestServiceLookupBoundAuthIndexAppliesBindingsWithAccountBindStrategy(t *testing.T) {
	const clientKey = "sk-client"

	service := &Service{
		cfg:         &config.Config{},
		coreManager: coreauth.NewManager(nil, nil, nil),
	}
	service.rebuildBindingMap(&config.Config{
		SDKConfig: config.SDKConfig{
			APIKeys:            config.FlexAPIKeyList{clientKey},
			APIKeyAuthBindings: map[string]string{clientKey: "idx-bound"},
		},
		Routing: internalconfig.RoutingConfig{
			Strategy:            "account-bind",
			DefaultModelAccount: "idx-default",
		},
	})

	if got, ok := service.LookupBoundAuthIndex(clientKey); !ok || got != "idx-bound" {
		t.Fatalf("binding not active in account-bind: got=%q ok=%v", got, ok)
	}
	if got, ok := service.LookupBoundAuthIndex("sk-other"); !ok || got != "idx-default" {
		t.Fatalf("default binding not active in account-bind: got=%q ok=%v", got, ok)
	}
}
