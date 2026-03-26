-- Composite index for profile search endpoint: filters by role, orders by created_at DESC
-- Covers the SearchPublic query: WHERE u.role = $1 ORDER BY u.created_at DESC LIMIT $3
CREATE INDEX IF NOT EXISTS idx_users_role_created_at ON users(role, created_at DESC);

-- Index for referrer search: filters on role + referrer_enabled, orders by created_at
CREATE INDEX IF NOT EXISTS idx_users_referrer_search ON users(created_at DESC)
    WHERE role = 'provider' AND referrer_enabled = true;
