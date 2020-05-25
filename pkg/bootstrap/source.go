package bootstrap

import envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

type ConfigSource = envoy_config_core_v3.ConfigSource

type ApiConfigSource = envoy_config_core_v3.ConfigSource_ApiConfigSource
type PathConfigSource = envoy_config_core_v3.ConfigSource_Path
type AdsConfigSource = envoy_config_core_v3.ConfigSource_Ads

// NewApiConfigSource returns a *ApiConfigSource for the named GRPC cluster.
func NewApiConfigSource(clusterName string) *ApiConfigSource {
	api := &ApiConfigSource{
		ApiConfigSource: &envoy_config_core_v3.ApiConfigSource{
			ApiType: envoy_config_core_v3.ApiConfigSource_GRPC,
			GrpcServices: []*envoy_config_core_v3.GrpcService{
				&envoy_config_core_v3.GrpcService{
					TargetSpecifier: &envoy_config_core_v3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &envoy_config_core_v3.GrpcService_EnvoyGrpc{
							ClusterName: clusterName,
						},
					},
				},
			},
			RefreshDelay:              nil,
			RequestTimeout:            nil,
			RateLimitSettings:         nil,
			SetNodeOnFirstMessageOnly: true,
		},
	}

	return api
}

// NewPathConfigSource ...
func NewPathConfigSource(path string) *PathConfigSource {
	return &PathConfigSource{
		Path: path,
	}
}

// NewAdsConfigSource ...
func NewAdsConfigSource() *AdsConfigSource {
	return &AdsConfigSource{
		Ads: &envoy_config_core_v3.AggregatedConfigSource{},
	}
}
