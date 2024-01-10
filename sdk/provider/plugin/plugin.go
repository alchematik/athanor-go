package plugin

import (
	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"

	"github.com/hashicorp/go-plugin"
)

type Plugin struct {
	plugin.Plugin

	server providerpb.ProviderServer
}
