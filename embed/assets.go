package embed

import (
	"time"

	"github.com/jessevdk/go-assets"
)

var _Assetseaff2d637ccc83b7ed4d76a0e29e5bc5bbbc0694 = "admin:\n  access_log_path: /dev/stdout\n  address:\n    socket_address:\n      address: 0.0.0.0\n      port_value: 9095\n\nstatic_resources:\n  listeners:\n    - name: ingress\n      address:\n        socket_address:\n          address: 0.0.0.0\n          port_value: 9090\n      filter_chains:\n        - filters:\n            - name: envoy.http_connection_manager\n              config:\n                access_log:\n                  name: envoy.file_access_log\n                  config:\n                    path: /dev/stdout\n                  filter:\n                    not_health_check_filter: {}\n                codec_type: auto\n                idle_timeout: 600s\n                stat_prefix: ingress_http\n                http_filters:\n                  - name: envoy.grpc_web\n                  - name: envoy.router\n                route_config:\n                  virtual_hosts:\n                    - name: envoy\n                      domains:\n                        - \"*\"\n                      routes:\n                        - match:\n                            prefix: /pipe.api.service.pipedservice.PipedService/\n                            grpc:\n                          route:\n                            cluster: api-piped-api\n                        - match:\n                            prefix: /pipe.api.service.webservice.WebService/\n                            grpc:\n                          route:\n                            cluster: api-web-api\n                        - match:\n                            path: /hook\n                          route:\n                            cluster: api-http\n                        - match:\n                            prefix: /auth\n                          route:\n                            cluster: api-http\n                        - match:\n                            prefix: /\n                          route:\n                            cluster: web\n  clusters:\n    - name: api-piped-api\n      http2_protocol_options: {}\n      connect_timeout: 0.25s\n      type: strict_dns\n      lb_policy: round_robin\n      load_assignment:\n        cluster_name: api-piped-api\n        endpoints:\n          - lb_endpoints:\n              - endpoint:\n                  address:\n                    socket_address:\n                      address: ADDRESS_API\n                      port_value: 9080\n    - name: api-web-api\n      http2_protocol_options: {}\n      connect_timeout: 0.25s\n      type: strict_dns\n      lb_policy: round_robin\n      load_assignment:\n        cluster_name: api-web-api\n        endpoints:\n          - lb_endpoints:\n              - endpoint:\n                  address:\n                    socket_address:\n                      address: ADDRESS_API\n                      port_value: 9081\n    - name: api-http\n      #http2_protocol_options: {}\n      connect_timeout: 0.25s\n      type: strict_dns\n      lb_policy: round_robin\n      load_assignment:\n        cluster_name: api-http\n        endpoints:\n          - lb_endpoints:\n              - endpoint:\n                  address:\n                    socket_address:\n                      address: ADDRESS_API\n                      port_value: 9082\n    - name: web\n      #http2_protocol_options: {}\n      connect_timeout: 0.25s\n      type: strict_dns\n      lb_policy: round_robin\n      load_assignment:\n        cluster_name: web\n        endpoints:\n          - lb_endpoints:\n              - endpoint:\n                  address:\n                    socket_address:\n                      address: ADDRESS_WEB\n                      port_value: 9082\n"

// Assets returns go-assets FileSystem
var Assets = assets.NewFileSystem(map[string][]string{"/": []string{"assets"}, "/assets": []string{"envoy-config.yaml"}}, map[string]*assets.File{
	"/assets": &assets.File{
		Path:     "/assets",
		FileMode: 0x800001ed,
		Mtime:    time.Unix(1605449609, 1605449609160049000),
		Data:     nil,
	}, "/assets/envoy-config.yaml": &assets.File{
		Path:     "/assets/envoy-config.yaml",
		FileMode: 0x1a4,
		Mtime:    time.Unix(1607150714, 1607150714882600213),
		Data:     []byte(_Assetseaff2d637ccc83b7ed4d76a0e29e5bc5bbbc0694),
	}, "/": &assets.File{
		Path:     "/",
		FileMode: 0x800001ed,
		Mtime:    time.Unix(1605449649, 1605449649617493000),
		Data:     nil,
	}}, "")
