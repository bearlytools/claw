package health

import (
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/client"
)

// Check performs a health check on the server.
// Use an empty string to check the overall server health.
// Returns the serving status and any error that occurred.
func Check(ctx context.Context, conn *client.Conn, service string) (ServingStatus, error) {
	sync, err := conn.Sync(ctx, "health", "Health", "Check")
	if err != nil {
		return Unknown, err
	}
	defer sync.Close()

	req := NewHealthCheckRequest(ctx).SetService(service)
	reqBytes, err := req.Marshal()
	if err != nil {
		return Unknown, err
	}

	respBytes, err := sync.Call(ctx, reqBytes)
	if err != nil {
		return Unknown, err
	}

	resp := NewHealthCheckResponse(ctx)
	if err := resp.Unmarshal(respBytes); err != nil {
		return Unknown, err
	}

	return resp.Status(), nil
}
