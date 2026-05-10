package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/port/service"
)

type JWTService struct {
	secret        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTService(secret string, accessExpiry, refreshExpiry time.Duration) *JWTService {
	return &JWTService{
		secret:        secret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// customClaims holds everything we pack into an access token.
//
// Organization context (OrgID, OrgRole) is optional and omitted from the
// JSON when the user is a solo Provider. The omitempty tags keep the
// token payload small for users without an org.
//
// SessionVersion is intentionally NOT marked omitempty: we want it to be
// explicitly present (even when 0) so the middleware can distinguish
// "claim present, value 0" from "claim absent, assume 0".
type customClaims struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role,omitempty"`
	IsAdmin bool   `json:"is_admin,omitempty"`
	Type    string `json:"type"`

	// Organization context for members of an organization. When absent,
	// the user is a solo Provider (or a not-yet-activated account).
	OrgID   string `json:"org_id,omitempty"`
	OrgRole string `json:"org_role,omitempty"`

	// Permissions is the effective permission set for this user's org
	// membership, resolved at issuance with the org's role overrides
	// applied. The auth middleware writes this into request context so
	// RequirePermission honors per-org customizations without querying
	// the database on the hot path.
	Permissions []string `json:"perms,omitempty"`

	// SessionVersion is the revocation anchor. Middleware compares this
	// against users.session_version and rejects on mismatch.
	SessionVersion int `json:"sv,omitempty"`

	// B.8 — refresh-token family lineage (refresh tokens only;
	// omitempty keeps access tokens unchanged).
	//
	// FamilyRootJTI is the JTI of the first token in this rotation
	// chain. ChainDepth is the number of rotations from that root.
	// FamilyRootIAT is the unix timestamp of the family-root token's
	// issuance — used to enforce the absolute family lifetime cap.
	FamilyRootJTI string `json:"frj,omitempty"`
	ChainDepth    int    `json:"cd,omitempty"`
	FamilyRootIAT int64  `json:"fiat,omitempty"`

	jwt.RegisteredClaims
}

func (s *JWTService) GenerateAccessToken(input service.AccessTokenInput) (string, error) {
	claims := customClaims{
		UserID:         input.UserID.String(),
		Role:           input.Role,
		IsAdmin:        input.IsAdmin,
		Type:           "access",
		OrgRole:        input.OrgRole,
		Permissions:    input.Permissions,
		SessionVersion: input.SessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	if input.OrganizationID != nil {
		claims.OrgID = input.OrganizationID.String()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	return s.GenerateRefreshTokenWithLineage(service.RefreshTokenInput{UserID: userID})
}

// GenerateRefreshTokenWithLineage mints a refresh token, copying
// family lineage forward when provided. On a fresh family
// (FamilyRootJTI empty) the new token's JTI is also the root: the
// family is born self-rooted, ChainDepth is 0, and FamilyRootIAT is
// "now". On a rotation, the caller passes the parent token's lineage
// + ChainDepth + 1 and we copy them verbatim.
func (s *JWTService) GenerateRefreshTokenWithLineage(input service.RefreshTokenInput) (string, error) {
	now := time.Now()
	jti := uuid.New().String()

	familyRootJTI := input.FamilyRootJTI
	if familyRootJTI == "" {
		// Fresh family — this token is its own root.
		familyRootJTI = jti
	}
	familyRootIAT := input.FamilyRootIAT
	if familyRootIAT.IsZero() {
		familyRootIAT = now
	}

	claims := customClaims{
		UserID:        input.UserID.String(),
		Type:          "refresh",
		FamilyRootJTI: familyRootJTI,
		ChainDepth:    input.ChainDepth,
		FamilyRootIAT: familyRootIAT.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) ValidateAccessToken(tokenString string) (*service.TokenClaims, error) {
	return s.validateToken(tokenString, "access")
}

func (s *JWTService) ValidateRefreshToken(tokenString string) (*service.TokenClaims, error) {
	return s.validateToken(tokenString, "refresh")
}

func (s *JWTService) validateToken(tokenString string, expectedType string) (*service.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.Type != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.Type)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id in token: %w", err)
	}

	result := &service.TokenClaims{
		UserID:         userID,
		Role:           claims.Role,
		IsAdmin:        claims.IsAdmin,
		ExpiresAt:      claims.ExpiresAt.Time,
		JTI:            claims.ID,
		OrgRole:        claims.OrgRole,
		Permissions:    claims.Permissions,
		SessionVersion: claims.SessionVersion,
		FamilyRootJTI:  claims.FamilyRootJTI,
		ChainDepth:     claims.ChainDepth,
	}
	if claims.FamilyRootIAT > 0 {
		result.FamilyRootIAT = time.Unix(claims.FamilyRootIAT, 0).UTC()
	}
	if claims.OrgID != "" {
		orgID, err := uuid.Parse(claims.OrgID)
		if err != nil {
			return nil, fmt.Errorf("invalid org id in token: %w", err)
		}
		result.OrganizationID = &orgID
	}
	return result, nil
}
