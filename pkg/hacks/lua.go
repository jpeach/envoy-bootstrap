package hacks

import (
	"fmt"
	"strings"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_filters_http_lua_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
	"github.com/jpeach/envoy-bootstrap/pkg/must"
	"github.com/jpeach/envoy-bootstrap/pkg/xds"

	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

type HTTPConnectionManager = envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager
type HTTPFilter = envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter

func FilterMisdirectedRequests(fqdn string) *bootstrap.HTTPFilter {
	type Lua = envoy_extensions_filters_http_lua_v3.Lua

	code := `
function envoy_on_request(request_handle)
        local headers = request_handle:headers()
        local host = headers:get(":authority")
        local target = "%s"

        if host ~= target then
                s, e = string.find(host, ":", 1, true)
                if s ~= nil then
                        host = string.sub(host, 1, s - 1)
                end

                if host ~= target then
                        request_handle:respond(
                                {[":status"] = "421"},
                                string.format("misdirected request to %%q", headers:get(":authority"))
                        )
                end

        end
end
        `

	return bootstrap.NewHTTPFilter(
		"envoy.filters.http.lua",
		bootstrap.ProtoV2(&Lua{InlineCode: fmt.Sprintf(code, fqdn)}),
	)
}

// HackLuaFilter ...
func HackLuaFilter(spec Spec) xds.Snapshot {
	name := RandomStringN(10)
	addr := must.IP(spec.Parameters["address"].IP())
	port := must.Int64(spec.Parameters["port"].AsInt64())
	cluster := must.String(spec.Parameters["cluster"].AsString())
	count := must.Int64(spec.Parameters["count"].Or("1").AsInt64())

	if cluster == "" {
		cluster = fmt.Sprintf("lua/cluster/%d", port)
	}

	type RouteSpecifier = envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_Rds
	type RDS = envoy_extensions_filters_network_http_connection_manager_v3.Rds

	var filterChains []*bootstrap.FilterChain

	for i := int64(0); i < count; i++ {
		hostname := fmt.Sprintf("lua-%d.example.com", i)
		c := &bootstrap.FilterChain{
			FilterChainMatch: &bootstrap.FilterChainMatch{
				ServerNames: []string{hostname},
			},
			Filters: []*bootstrap.Filter{
				bootstrap.NewFilter(
					"envoy.filters.network.http_connection_manager",
					bootstrap.ProtoV2(&HTTPConnectionManager{
						// RouteSpecifier field is required.
						RouteSpecifier: &RouteSpecifier{
							Rds: &RDS{
								RouteConfigName: fmt.Sprintf("lua/route/%s", name),
								ConfigSource: &bootstrap.ConfigSource{
									ConfigSourceSpecifier: bootstrap.NewAdsConfigSource(),
									ResourceApiVersion:    envoy_config_core_v3.ApiVersion_V3,
								},
							},
						},
						// StatPrefix field is required.
						StatPrefix: strings.Replace(hostname, ".", "-", -1),
						HttpFilters: []*bootstrap.HTTPFilter{
							FilterMisdirectedRequests(hostname),
							&bootstrap.HTTPFilter{
								Name: "envoy.filters.http.gzip",
							},
							&bootstrap.HTTPFilter{
								Name: "envoy.filters.http.grpc_web",
							},
							&bootstrap.HTTPFilter{
								Name: "envoy.filters.http.router",
							},
						},
					}),
				),
			},
		}

		filterChains = append(filterChains, c)
	}

	listener := &bootstrap.Listener{
		Name: "hack/lua/listener",
		Address: bootstrap.NewSocketAddress(
			&bootstrap.SocketAddress{
				Protocol:      bootstrap.TCP,
				Address:       addr.String(),
				PortSpecifier: bootstrap.NewPortValue(uint32(port)),
				Ipv4Compat:    false,
			}),
		FilterChains: filterChains,
		ListenerFilters: []*bootstrap.ListenerFilter{
			&bootstrap.ListenerFilter{
				Name: "envoy.filters.listener.tls_inspector",
			},
		},
		ListenerFiltersTimeout: ptypes.DurationProto(time.Second * 15), // Default.
		AccessLog:              nil,
	}

	snap := xds.Snapshot{}
	snap.Resources[xds.ListenerType] = xds.NewResources(NewVersion(), listener)

	return snap
}
