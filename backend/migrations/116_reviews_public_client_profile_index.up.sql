-- 116_reviews_public_client_profile_index.up.sql
--
-- Mirror of the existing idx_reviews_public_profile index (082) but for
-- the provider→client side of the double-blind pair. The public client
-- profile endpoint filters on reviewed_organization_id +
-- side='provider_to_client' + published_at IS NOT NULL, same shape as the
-- provider side; adding the partial index here keeps the read plan an
-- index-only scan even as the reviews table grows.

CREATE INDEX IF NOT EXISTS idx_reviews_public_client_profile
    ON reviews(reviewed_organization_id, published_at DESC)
    WHERE side = 'provider_to_client' AND published_at IS NOT NULL;
