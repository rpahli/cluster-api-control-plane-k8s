package v1beta3

import (
	"net"
	"strconv"

	netutils "k8s.io/utils/net"

	"github.com/pkg/errors"
)

// APIEndpointFromString returns an APIEndpoint struct based on a "host:port" raw string.
func APIEndpointFromString(apiEndpoint string) (APIEndpoint, error) {
	apiEndpointHost, apiEndpointPortStr, err := net.SplitHostPort(apiEndpoint)
	if err != nil {
		return APIEndpoint{}, errors.Wrapf(err, "invalid advertise address endpoint: %s", apiEndpoint)
	}
	if netutils.ParseIPSloppy(apiEndpointHost) == nil {
		return APIEndpoint{}, errors.Errorf("invalid API endpoint IP: %s", apiEndpointHost)
	}
	apiEndpointPort, err := net.LookupPort("tcp", apiEndpointPortStr)
	if err != nil {
		return APIEndpoint{}, errors.Wrapf(err, "invalid advertise address endpoint port: %s", apiEndpointPortStr)
	}
	return APIEndpoint{
		AdvertiseAddress: apiEndpointHost,
		BindPort:         int32(apiEndpointPort),
	}, nil
}

func (endpoint *APIEndpoint) String() string {
	return net.JoinHostPort(endpoint.AdvertiseAddress, strconv.FormatInt(int64(endpoint.BindPort), 10))
}
