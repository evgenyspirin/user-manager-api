package rest

const (
	// api
	RouteApiV1 = "/api/v1"

	// auth
	RouteAuth  = RouteApiV1 + "/auth"
	RouteLogin = RouteAuth + "/login"

	RouteUsers     = RouteApiV1 + "/users"
	RouteUser      = RouteUsers + "/:user_id"
	RouteUserFiles = RouteUser + "/files"

	// ops
	RouteHealth  = RouteApiV1 + "/healthz"
	RouteMetrics = RouteApiV1 + "/metrics"
)
