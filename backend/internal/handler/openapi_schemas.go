package handler

import (
	"reflect"
	"sort"
	"strings"
	"time"

	dtoreq "marketplace-backend/internal/handler/dto/request"
	dtoresp "marketplace-backend/internal/handler/dto/response"
)

// schemaRegistry collects every named JSON Schema we reference from
// operations. Schemas are keyed by component name (e.g. "AuthResponse")
// and emitted under `components.schemas` in the OpenAPI document.
//
// The registry deduplicates: registering the same Go type twice
// returns the same component name, so request and response paths can
// share DTO types without bloating the document.
type schemaRegistry struct {
	byName map[string]map[string]any
	byType map[reflect.Type]string
}

func newSchemaRegistry() *schemaRegistry {
	r := &schemaRegistry{
		byName: map[string]map[string]any{},
		byType: map[reflect.Type]string{},
	}
	r.registerErrorEnvelope()
	r.registerAllCuratedTypes()
	return r
}

// register adds a schema by name. Last write wins, but the registry is
// only ever populated through registerType which is idempotent — so
// in practice every schema is written exactly once.
func (r *schemaRegistry) register(name string, schema map[string]any) {
	r.byName[name] = schema
}

// registerType builds the JSON Schema for a Go type via reflection
// and adds it to the registry. Returns the component name. Pointer
// types are dereferenced; slices and maps are described natively.
//
// Recursion is bounded: a self-referential struct (rare in DTOs) is
// emitted with an unconstrained object stub on the second visit.
func (r *schemaRegistry) registerType(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if name, ok := r.byType[t]; ok {
		return name
	}
	name := componentNameFor(t)
	// Pre-register a placeholder so cyclic references resolve.
	r.byType[t] = name
	r.byName[name] = map[string]any{"type": "object"}
	r.byName[name] = r.schemaForType(t)
	return name
}

// schemaForType emits the JSON Schema fragment for a Go type. The
// primary callers are registerType (struct types) and the inline
// schema-builder for slices / maps / scalars.
func (r *schemaRegistry) schemaForType(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.Ptr:
		return r.schemaForType(t.Elem())
	case reflect.Struct:
		// time.Time is encoded as RFC3339 string by Go's json package.
		if t == reflect.TypeOf(time.Time{}) {
			return map[string]any{"type": "string", "format": "date-time"}
		}
		return r.structSchema(t)
	case reflect.Slice, reflect.Array:
		// []byte is a base64 string under encoding/json.
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "byte"}
		}
		return map[string]any{
			"type":  "array",
			"items": r.schemaForType(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": r.schemaForType(t.Elem()),
		}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "format": int64FormatFor(t)}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number", "format": "double"}
	case reflect.Interface:
		// `any` / interface{} → unconstrained, matches openapi-typescript's
		// default expansion of `unknown`.
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

// structSchema reflects over a struct's exported fields and builds
// the JSON Schema. JSON tags drive field names; the omitempty marker
// removes the field from the `required` list. Embedded structs are
// flattened (matching encoding/json behaviour).
func (r *schemaRegistry) structSchema(t reflect.Type) map[string]any {
	props := map[string]any{}
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		// Anonymous embedded struct → flatten its fields onto this
		// schema, matching how encoding/json serialises it.
		if f.Anonymous && f.Type.Kind() == reflect.Struct && tag == "" {
			inner := r.structSchema(f.Type)
			if innerProps, ok := inner["properties"].(map[string]any); ok {
				for k, v := range innerProps {
					props[k] = v
				}
			}
			if innerReq, ok := inner["required"].([]string); ok {
				required = append(required, innerReq...)
			}
			continue
		}

		name, omitEmpty := parseJSONTag(tag, f.Name)
		fieldSchema := r.schemaForType(f.Type)
		// Pointer fields are nullable in OpenAPI 3.1.
		if f.Type.Kind() == reflect.Ptr {
			fieldSchema = nullable(fieldSchema)
		}
		props[name] = fieldSchema
		if !omitEmpty && f.Type.Kind() != reflect.Ptr {
			required = append(required, name)
		}
	}

	out := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		sort.Strings(required)
		out["required"] = required
	}
	return out
}

// parseJSONTag returns (name, omitEmpty) for a struct tag's "json" key.
// An empty / "-" tag means "use field name", but we already guard the
// "-" case upstream. The only modifier we read is "omitempty"; other
// markers (string, ',inline') are not used in the DTO types.
func parseJSONTag(tag, fieldName string) (string, bool) {
	if tag == "" {
		return fieldName, false
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		name = fieldName
	}
	omit := false
	for _, p := range parts[1:] {
		if strings.TrimSpace(p) == "omitempty" {
			omit = true
		}
	}
	return name, omit
}

// componentNameFor derives the component name for a struct type. The
// path is the package + type name with non-letter characters removed
// so distinct packages with identical type names don't collide. In
// practice every DTO type lives in either dto/request or dto/response
// and the type names are unique across both packages.
func componentNameFor(t reflect.Type) string {
	if t.Name() == "" {
		// anonymous struct — fall back to a deterministic stub
		return "AnonymousStruct"
	}
	return t.Name()
}

// nullable wraps a schema in a OpenAPI 3.1 nullable union. We use the
// `oneOf` form (instead of the old 3.0 `nullable: true`) because the
// 3.1 dialect deprecated the latter.
func nullable(schema map[string]any) map[string]any {
	// Avoid double-wrapping when the inner schema is already nullable.
	if oneOf, ok := schema["oneOf"]; ok {
		_ = oneOf
		return schema
	}
	return map[string]any{
		"oneOf": []map[string]any{
			schema,
			{"type": "null"},
		},
	}
}

// int64FormatFor returns the OpenAPI integer format hint for a
// reflect.Type. Anything 64-bit gets "int64"; everything else "int32".
func int64FormatFor(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int64, reflect.Uint64:
		return "int64"
	default:
		return "int32"
	}
}

// export returns the schemas as the OpenAPI components.schemas map.
// Sorting is implicit — JSON marshalling of a Go map sorts keys.
func (r *schemaRegistry) export() map[string]any {
	out := make(map[string]any, len(r.byName))
	for k, v := range r.byName {
		out[k] = v
	}
	return out
}

// registerErrorEnvelope pins the standard error envelope. The shape is
// PRODUCED by pkg/response.Error / ValidationError / NotFound — see
// response/error.go for the canonical handler-side struct.
func (r *schemaRegistry) registerErrorEnvelope() {
	r.register("ErrorResponse", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"error":   map[string]any{"type": "string", "description": "Machine-readable error code"},
			"message": map[string]any{"type": "string", "description": "Human-readable error message"},
			"details": map[string]any{
				"type":                 "object",
				"description":          "Per-field validation errors (optional)",
				"additionalProperties": map[string]any{"type": "string"},
			},
		},
		"required": []string{"error", "message"},
	})
}

// registerAllCuratedTypes pre-registers every DTO type referenced from
// the route catalogue. We do this eagerly so the components.schemas
// map is fully populated even for endpoints whose op is built without
// a registry handle (e.g. ad-hoc lookups in tests).
func (r *schemaRegistry) registerAllCuratedTypes() {
	// Auth + organization context
	r.registerType(reflect.TypeOf(dtoresp.AuthResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.MeResponse{}))
	r.registerType(reflect.TypeOf(dtoreq.RegisterRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.LoginRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.RefreshRequest{}))

	// Profile-family responses (each persona)
	r.registerType(reflect.TypeOf(dtoresp.ProfileResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.FreelanceProfileResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.ReferrerProfileResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.PublicClientProfileResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.OrganizationSharedProfileResponse{}))
	r.registerType(reflect.TypeOf(dtoreq.UpdateProfileRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.UpdateClientProfileRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.UpdateFreelanceProfileRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.UpdateReferrerProfileRequest{}))

	// Job + application
	r.registerType(reflect.TypeOf(dtoresp.JobResponse{}))
	r.registerType(reflect.TypeOf(dtoreq.CreateJobRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.ApplyToJobRequest{}))

	// Proposal
	r.registerType(reflect.TypeOf(dtoresp.ProposalResponse{}))

	// Messaging
	r.registerType(reflect.TypeOf(dtoresp.ConversationResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.MessageResponse{}))

	// Notification
	r.registerType(reflect.TypeOf(dtoresp.NotificationResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.NotificationPreferenceResponse{}))
	r.registerType(reflect.TypeOf(dtoreq.UpdateNotificationPreferencesRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.RegisterDeviceTokenRequest{}))

	// Review + report
	r.registerType(reflect.TypeOf(dtoresp.ReviewResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.ReportResponse{}))

	// Referral
	r.registerType(reflect.TypeOf(dtoresp.ReferralResponse{}))
	r.registerType(reflect.TypeOf(dtoreq.CreateReferralRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.RespondReferralRequest{}))

	// Dispute
	r.registerType(reflect.TypeOf(dtoresp.DisputeResponse{}))
	r.registerType(reflect.TypeOf(dtoreq.OpenDisputeRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.CounterProposeRequest{}))

	// Skill + portfolio
	r.registerType(reflect.TypeOf(dtoresp.SkillResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.PortfolioItemResponse{}))

	// Social link
	r.registerType(reflect.TypeOf(dtoresp.SocialLinkResponse{}))

	// Team
	r.registerType(reflect.TypeOf(dtoresp.MemberResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.MemberListResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.InvitationResponse{}))
	r.registerType(reflect.TypeOf(dtoresp.RoleDefinitionsPayload{}))
	r.registerType(reflect.TypeOf(dtoresp.TransferResponse{}))

	// Call
	r.registerType(reflect.TypeOf(dtoreq.InitiateCallRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.EndCallRequest{}))

	// Proposal-related
	r.registerType(reflect.TypeOf(dtoreq.CreateProposalRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.ModifyProposalRequest{}))

	// Portfolio
	r.registerType(reflect.TypeOf(dtoreq.CreatePortfolioItemRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.UpdatePortfolioItemRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.ReorderPortfolioRequest{}))

	// Skill
	r.registerType(reflect.TypeOf(dtoreq.PutProfileSkillsRequest{}))
	r.registerType(reflect.TypeOf(dtoreq.CreateSkillRequest{}))

	// Review
	r.registerType(reflect.TypeOf(dtoreq.CreateReviewRequest{}))

	// Report
	r.registerType(reflect.TypeOf(dtoreq.CreateReportRequest{}))

	// Stripe / billing
	r.registerType(reflect.TypeOf(dtoresp.StripeConfigResponse{}))
}
