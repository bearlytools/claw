package reflection

import (
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/client"
)

// ListServices retrieves all registered packages and services from the server.
func ListServices(ctx context.Context, conn *client.Conn) ([]PackageInfo, error) {
	sync, err := conn.Sync(ctx, "reflection", "Reflection", "ListServices")
	if err != nil {
		return nil, err
	}
	defer sync.Close()

	req := NewListServicesRequest(ctx)
	reqBytes, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	respBytes, err := sync.Call(ctx, reqBytes)
	if err != nil {
		return nil, err
	}

	resp := NewListServicesResponse(ctx)
	if err := resp.Unmarshal(respBytes); err != nil {
		return nil, err
	}

	// Collect packages into a slice.
	var packages []PackageInfo
	for i := 0; i < resp.PackagesLen(ctx); i++ {
		packages = append(packages, resp.PackagesGet(ctx, i))
	}

	return packages, nil
}

// GetServiceInfo retrieves information about a specific service.
// Returns the service info, whether it was found, and any error.
func GetServiceInfo(ctx context.Context, conn *client.Conn, pkg, service string) (ServiceInfo, bool, error) {
	sync, err := conn.Sync(ctx, "reflection", "Reflection", "GetServiceInfo")
	if err != nil {
		return ServiceInfo{}, false, err
	}
	defer sync.Close()

	req := NewGetServiceInfoRequest(ctx).SetPackage(pkg).SetService(service)
	reqBytes, err := req.Marshal()
	if err != nil {
		return ServiceInfo{}, false, err
	}

	respBytes, err := sync.Call(ctx, reqBytes)
	if err != nil {
		return ServiceInfo{}, false, err
	}

	resp := NewGetServiceInfoResponse(ctx)
	if err := resp.Unmarshal(respBytes); err != nil {
		return ServiceInfo{}, false, err
	}

	return resp.Service(), resp.Found(), nil
}

// GetMethodInfo retrieves information about a specific method.
// Returns the method info, whether it was found, and any error.
func GetMethodInfo(ctx context.Context, conn *client.Conn, pkg, service, method string) (MethodInfo, bool, error) {
	sync, err := conn.Sync(ctx, "reflection", "Reflection", "GetMethodInfo")
	if err != nil {
		return MethodInfo{}, false, err
	}
	defer sync.Close()

	req := NewGetMethodInfoRequest(ctx).SetPackage(pkg).SetService(service).SetMethod(method)
	reqBytes, err := req.Marshal()
	if err != nil {
		return MethodInfo{}, false, err
	}

	respBytes, err := sync.Call(ctx, reqBytes)
	if err != nil {
		return MethodInfo{}, false, err
	}

	resp := NewGetMethodInfoResponse(ctx)
	if err := resp.Unmarshal(respBytes); err != nil {
		return MethodInfo{}, false, err
	}

	return resp.Method(), resp.Found(), nil
}
