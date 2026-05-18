package admin

import (
	"context"

	"github.com/google/uuid"
)

// ActorSearchIndexer is the narrow port the admin moderation flow uses
// to keep the Typesense actor index consistent with a user's
// moderation status. It is deliberately tiny (ISP): the admin feature
// only ever needs to *remove* an actor (suspend / ban) or *reindex*
// it (unsuspend / unban → the actor is active again).
//
// Defined locally in the admin app package to avoid a cross-feature
// import of the search engine. The concrete implementation is a thin
// adapter that wraps the existing *searchindex.Publisher so the actor
// document keying (organization_id, every persona variant) stays
// IDENTICAL to the rest of the indexing pipeline — admin moderation
// never re-implements document serialization or ID derivation.
//
// Optional dependency: a nil ActorSearchIndexer makes every call a
// no-op so the admin service stays bootable without the search engine
// (tests, minimal deployments). Failures are NEVER propagated to the
// admin action — the DB status flip is the source of truth; search
// drift is logged and reconciled by the outbox worker.
type ActorSearchIndexer interface {
	// RemoveActor deindexes the organization's actor document(s) from
	// the search collection. Idempotent — removing an already-absent
	// document is a success.
	RemoveActor(ctx context.Context, orgID uuid.UUID) error

	// ReindexActor re-upserts the organization's actor document(s) so
	// the actor reappears in the public directories. Idempotent —
	// upsert by id never duplicates.
	ReindexActor(ctx context.Context, orgID uuid.UUID) error
}
