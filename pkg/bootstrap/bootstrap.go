package bootstrap

import (
	"os"

	"github.com/jpeach/envoy-bootstrap/pkg/must"

	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
)

func NewBootstrap() *Bootstrap {
	b := Bootstrap{
		Node: &Node{
			Id:                               must.String(os.Hostname()),
			Cluster:                          must.String(os.Hostname()),
			Metadata:                          nil,
			Locality:                          nil,
		},
		Admin: &Admin{
			AccessLogPath:        "/dev/null",
			Address: NewPipeAddress(&PipeAddress{
				Path:                 "/var/run/envoy/admin.pipe",
				Mode:                 0644,
			}),
		},
		StaticResources: &envoy_config_bootstrap_v3.Bootstrap_StaticResources{
			Listeners:            nil,
			Clusters:             nil,
			Secrets:              nil,
		},
	}

	return &b
}
