package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/port/service"
)

type SessionService struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewSessionService(client *goredis.Client, ttl time.Duration) *SessionService {
	return &SessionService{client: client, ttl: ttl}
}

// sessionData is the JSON payload stored in Redis for a session.
// Organization fields use pointer/empty to gracefully degrade for Providers.
type sessionData struct {
	UserID         string    `json:"user_id"`
	Role           string    `json:"role"`
	IsAdmin        bool      `json:"is_admin"`
	OrgID          string    `json:"org_id,omitempty"`
	OrgRole        string    `json:"org_role,omitempty"`
	SessionVersion int       `json:"sv,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *SessionService) Create(ctx context.Context, input service.CreateSessionInput) (*service.Session, error) {
	id := uuid.New().String()
	now := time.Now()

	data := sessionData{
		UserID:         input.UserID.String(),
		Role:           input.Role,
		IsAdmin:        input.IsAdmin,
		OrgRole:        input.OrgRole,
		SessionVersion: input.SessionVersion,
		CreatedAt:      now,
	}
	if input.OrganizationID != nil {
		data.OrgID = input.OrganizationID.String()
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal session: %w", err)
	}

	if err := s.client.Set(ctx, "session:"+id, payload, s.ttl).Err(); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}

	return &service.Session{
		ID:             id,
		UserID:         input.UserID,
		Role:           input.Role,
		IsAdmin:        input.IsAdmin,
		OrganizationID: input.OrganizationID,
		OrgRole:        input.OrgRole,
		SessionVersion: input.SessionVersion,
		CreatedAt:      now,
	}, nil
}

func (s *SessionService) Get(ctx context.Context, sessionID string) (*service.Session, error) {
	val, err := s.client.Get(ctx, "session:"+sessionID).Result()
	if err == goredis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	var data sessionData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	session := &service.Session{
		ID:             sessionID,
		UserID:         userID,
		Role:           data.Role,
		IsAdmin:        data.IsAdmin,
		OrgRole:        data.OrgRole,
		SessionVersion: data.SessionVersion,
		CreatedAt:      data.CreatedAt,
	}
	if data.OrgID != "" {
		orgID, err := uuid.Parse(data.OrgID)
		if err != nil {
			return nil, fmt.Errorf("parse org id: %w", err)
		}
		session.OrganizationID = &orgID
	}
	return session, nil
}

func (s *SessionService) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, "session:"+sessionID).Err()
}

// DeleteByUserID scans all session keys and deletes those belonging to the
// given user. Sessions are stored as "session:{uuid}" with a JSON body
// containing user_id. This uses SCAN to avoid blocking Redis on large keyspaces.
func (s *SessionService) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	var cursor uint64
	target := userID.String()

	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, "session:*", 100).Result()
		if err != nil {
			return fmt.Errorf("scan sessions: %w", err)
		}

		for _, key := range keys {
			val, err := s.client.Get(ctx, key).Result()
			if err != nil {
				continue // key expired between scan and get
			}
			var data sessionData
			if err := json.Unmarshal([]byte(val), &data); err != nil {
				continue
			}
			if data.UserID == target {
				_ = s.client.Del(ctx, key).Err()
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// CreateWSToken generates a short-lived token for WebSocket authentication.
// The token maps to the user's ID and expires in 60 seconds.
// This avoids exposing the session_id in a non-httpOnly cookie.
func (s *SessionService) CreateWSToken(ctx context.Context, userID uuid.UUID) (string, error) {
	token := uuid.New().String()
	key := "ws_token:" + token

	if err := s.client.Set(ctx, key, userID.String(), 60*time.Second).Err(); err != nil {
		return "", fmt.Errorf("store ws token: %w", err)
	}

	return token, nil
}

// ValidateWSToken validates a short-lived WS token and returns the user ID.
// The token is deleted after validation (single-use).
func (s *SessionService) ValidateWSToken(ctx context.Context, token string) (uuid.UUID, error) {
	key := "ws_token:" + token

	val, err := s.client.GetDel(ctx, key).Result()
	if err == goredis.Nil {
		return uuid.UUID{}, fmt.Errorf("ws token not found or expired")
	}
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("validate ws token: %w", err)
	}

	return uuid.Parse(val)
}
