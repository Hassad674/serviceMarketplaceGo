package postgres

const queryInsertPortfolioItem = `
INSERT INTO portfolio_items (id, user_id, title, description, link_url, position, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const queryInsertPortfolioMedia = `
INSERT INTO portfolio_media (id, portfolio_item_id, media_url, media_type, thumbnail_url, position, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

const queryGetPortfolioItemByID = `
SELECT id, user_id, title, description, link_url, position, created_at, updated_at
FROM portfolio_items
WHERE id = $1`

const queryListPortfolioMediaByItemID = `
SELECT id, portfolio_item_id, media_url, media_type, thumbnail_url, position, created_at
FROM portfolio_media
WHERE portfolio_item_id = $1
ORDER BY position ASC`

const queryListPortfolioByUserFirst = `
SELECT id, user_id, title, description, link_url, position, created_at, updated_at
FROM portfolio_items
WHERE user_id = $1
ORDER BY position ASC, created_at DESC
LIMIT $2`

const queryListPortfolioByUserWithCursor = `
SELECT id, user_id, title, description, link_url, position, created_at, updated_at
FROM portfolio_items
WHERE user_id = $1
  AND (position, id) > ($2, $3)
ORDER BY position ASC, created_at DESC
LIMIT $4`

const queryListMediaByItemIDs = `
SELECT id, portfolio_item_id, media_url, media_type, thumbnail_url, position, created_at
FROM portfolio_media
WHERE portfolio_item_id = ANY($1)
ORDER BY portfolio_item_id, position ASC`

const queryUpdatePortfolioItem = `
UPDATE portfolio_items
SET title = $2, description = $3, link_url = $4, updated_at = $5
WHERE id = $1`

const queryDeletePortfolioItem = `
DELETE FROM portfolio_items WHERE id = $1`

const queryCountPortfolioByUser = `
SELECT COUNT(*) FROM portfolio_items WHERE user_id = $1`

const queryDeleteMediaByItemID = `
DELETE FROM portfolio_media WHERE portfolio_item_id = $1`

const queryUpdatePortfolioPosition = `
UPDATE portfolio_items SET position = $2, updated_at = now()
WHERE id = $1 AND user_id = $3`
