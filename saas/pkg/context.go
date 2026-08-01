package saas

import "context"

// contextKey 上下文键类型
type contextKey string

const tenantContextKey contextKey = "saas_tenant_id"

// WithTenant 将租户ID存入上下文
func WithTenant(ctx context.Context, tenantID uint64) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}

// GetTenantFromContext 从上下文获取租户ID
func GetTenantFromContext(ctx context.Context) (uint64, bool) {
	val := ctx.Value(tenantContextKey)
	if val == nil {
		return 0, false
	}
	tenantID, ok := val.(uint64)
	return tenantID, ok
}
