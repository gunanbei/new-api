package jsplugin

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const APIVersion1 = 1

type LocalizedText map[string]string

func (t LocalizedText) MarshalJSON() ([]byte, error) { return common.Marshal(map[string]string(t)) }
func (t *LocalizedText) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "" {
		*t = nil
		return nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := common.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = LocalizedText{"en": s}
		return nil
	}
	var values map[string]string
	if err := common.Unmarshal(data, &values); err != nil {
		return err
	}
	*t = LocalizedText(values)
	return nil
}

type AuthorMeta struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}
type AuthMeta struct {
	Type string `json:"type"`
}
type UsageFieldSchema struct {
	Type        string        `json:"type,omitempty"`
	Unit        string        `json:"unit,omitempty"`
	Enum        []string      `json:"enum,omitempty"`
	Description LocalizedText `json:"description,omitempty"`
}
type UsageExample struct {
	Label string         `json:"label"`
	Facts map[string]any `json:"facts"`
}

type Meta struct {
	APIVersion    int                         `json:"apiVersion"`
	Key           string                      `json:"key"`
	Name          string                      `json:"name"`
	Icon          string                      `json:"icon,omitempty"`
	Description   LocalizedText               `json:"description,omitempty"`
	Version       string                      `json:"version"`
	Author        AuthorMeta                  `json:"author"`
	ChannelTypes  []int                       `json:"channelTypes,omitempty"`
	Models        []string                    `json:"models"`
	FetchMode     string                      `json:"fetchMode"`
	AllowedHosts  []string                    `json:"allowedHosts,omitempty"`
	Routes        []Route                     `json:"routes,omitempty"`
	Protocols     []ProtocolClaim             `json:"protocols,omitempty"`
	UsageSchema   map[string]UsageFieldSchema `json:"usageSchema,omitempty"`
	UsageExamples []UsageExample              `json:"usageExamples,omitempty"`
	Auth          AuthMeta                    `json:"auth,omitempty"`
}

func (m Meta) ProtocolSupports(protocol, mode string) bool {
	for _, p := range m.Protocols {
		if p.Name == protocol {
			return slices.Contains(p.Supports, mode)
		}
	}
	return false
}

type RouteType string

const (
	RouteTypeSubmit  RouteType = "submit"
	RouteTypeQuery   RouteType = "query"
	RouteTypeDynamic RouteType = "dynamic"
)

type Route struct {
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	Type        RouteType `json:"type"`
	Action      string    `json:"action,omitempty"`
	Decode      string    `json:"decode,omitempty"`
	Render      string    `json:"render,omitempty"`
	TaskIDParam string    `json:"taskIdParam,omitempty"`
	Models      []string  `json:"models,omitempty"`
}
type ProtocolClaim struct {
	Name       string   `json:"name"`
	Models     []string `json:"models,omitempty"`
	Supports   []string `json:"supports,omitempty"`
	objectForm bool
}
type ProtocolMode struct {
	Name string
	Hook string
}
type BodyKind string

const (
	BodyNone      BodyKind = "none"
	BodyJSON      BodyKind = "json"
	BodyForm      BodyKind = "form"
	BodyMultipart BodyKind = "multipart"
)

type HostProtocolOperation struct {
	Name                    string
	Methods                 []string
	Path                    string
	BodyKinds               []BodyKind
	ModelField              string
	RequiredProtocolMembers []string
	Modes                   []ProtocolMode
	RequiredDriverHooks     []string
}
type HostProtocolDefinition struct {
	Name       string
	Operations []HostProtocolOperation
}

var hostProtocols = []HostProtocolDefinition{
	{Name: "openai_responses", Operations: []HostProtocolOperation{{Name: "create", Methods: []string{http.MethodPost}, Path: "/v1/responses", BodyKinds: []BodyKind{BodyJSON}, ModelField: "model", RequiredProtocolMembers: []string{"decodeRequest"}, Modes: []ProtocolMode{{Name: "stream", Hook: "renderEvents"}, {Name: "sync", Hook: "renderFinal"}, {Name: "background", Hook: "renderFinal"}}}, {Name: "retrieve", Methods: []string{http.MethodGet}, Path: "/v1/responses/:response_id", BodyKinds: []BodyKind{BodyNone}}}},
	{Name: "openai_video", Operations: []HostProtocolOperation{{Name: "create", Methods: []string{http.MethodPost}, Path: "/v1/videos", BodyKinds: []BodyKind{BodyJSON, BodyMultipart}, ModelField: "model", RequiredProtocolMembers: []string{"decodeRequest"}}, {Name: "retrieve", Methods: []string{http.MethodGet}, Path: "/v1/videos/:task_id", BodyKinds: []BodyKind{BodyNone}, RequiredProtocolMembers: []string{"render"}}, {Name: "content", Methods: []string{http.MethodGet, http.MethodHead}, Path: "/v1/videos/:task_id/content", BodyKinds: []BodyKind{BodyNone}, RequiredDriverHooks: []string{"listArtifacts", "buildContentRequest"}}}},
}

func HostProtocol(name string) (HostProtocolDefinition, bool) {
	for _, p := range hostProtocols {
		if p.Name == name {
			return p, true
		}
	}
	return HostProtocolDefinition{}, false
}
func HostProtocols() []HostProtocolDefinition {
	return append([]HostProtocolDefinition(nil), hostProtocols...)
}
func (d HostProtocolDefinition) DefinedModes() []ProtocolMode {
	var out []ProtocolMode
	seen := map[string]bool{}
	for _, o := range d.Operations {
		for _, m := range o.Modes {
			if !seen[m.Name] {
				seen[m.Name] = true
				out = append(out, m)
			}
		}
	}
	return out
}

type LoadedPlugin struct {
	Meta   Meta
	Engine *Engine
}

type FileReference struct {
	Ref, Field, Filename, MimeType string
	Size                           int64
}
type FilePlaceholder struct {
	FileRef  string `json:"__fileRef"`
	Encoding string `json:"encoding"`
	MimeType string `json:"mimeType,omitempty"`
	MaxBytes int64  `json:"maxBytes,omitempty"`
}
type DecodedBody struct {
	Kind   string
	Value  any
	Fields map[string][]string
	Files  []FileReference
}
type SubmitIntent struct {
	Kind          string   `json:"kind"`
	Model         string   `json:"model"`
	Action        string   `json:"action,omitempty"`
	RequestBody   any      `json:"requestBody,omitempty"`
	OriginTaskIDs []string `json:"originTaskIds,omitempty"`
}
type QueryIntent struct {
	Kind    string   `json:"kind"`
	TaskIDs []string `json:"taskIds"`
}
type TaskView struct {
	TaskID     string         `json:"task_id"`
	Status     string         `json:"status"`
	Progress   string         `json:"progress,omitempty"`
	FailReason string         `json:"fail_reason,omitempty"`
	CreatedAt  int64          `json:"created_at,omitempty"`
	UpdatedAt  int64          `json:"updated_at,omitempty"`
	Data       any            `json:"data,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}
type DriverContext struct {
	RequestBody                                                             any               `json:"requestBody"`
	RequestHeaders                                                          map[string]string `json:"requestHeaders"`
	Action, Model, UpstreamModel, BaseURL, APIKey, AuthHeader, PublicTaskID string
	Files                                                                   []FileReference
	OriginTasks                                                             []OriginTaskView
}
type OriginTaskView struct {
	TaskID, UpstreamTaskID, Action, Status string
	Data                                   any
}
type RequestDescriptor struct {
	URL                         string            `json:"url"`
	Method                      string            `json:"method,omitempty"`
	Headers                     map[string]string `json:"headers,omitempty"`
	Body                        any               `json:"body,omitempty"`
	Credentialless              bool              `json:"credentialless,omitempty"`
	Action, Model, RewriteModel string
	BodyType                    string        `json:"bodyType,omitempty"`
	Parts                       []RequestPart `json:"parts,omitempty"`
}
type RequestPart struct {
	Name     string `json:"name"`
	Value    any    `json:"value,omitempty"`
	FileRef  string `json:"fileRef,omitempty"`
	Filename string `json:"filename,omitempty"`
}
type UpstreamResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       any                 `json:"body"`
}
type NormalizedTaskResult struct {
	TaskID           string  `json:"taskId,omitempty"`
	Status           string  `json:"status"`
	Progress         string  `json:"progress,omitempty"`
	Reason           string  `json:"reason,omitempty"`
	URL              string  `json:"url,omitempty"`
	RemoteURL        string  `json:"remoteUrl,omitempty"`
	CompletionTokens float64 `json:"completionTokens,omitempty"`
	TotalTokens      float64 `json:"totalTokens,omitempty"`
}
type TaskArtifact struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	MimeType string `json:"mimeType,omitempty"`
}

var pluginKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
var pluginVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
var localeTagPattern = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$`)

// CompilePlugin compiles source and validates the v1 host contract without publishing it.
func CompilePlugin(source string, options Options) (*LoadedPlugin, error) {
	engine, err := Compile(source, options)
	if err != nil {
		return nil, err
	}
	value, err := engine.Export(context.Background(), "meta")
	if err != nil {
		return nil, err
	}
	meta, err := decodeMeta(value)
	if err != nil {
		return nil, err
	}
	if err = ValidateV1Meta(meta); err != nil {
		return nil, err
	}
	engine.key, engine.version = meta.Key, meta.Version
	required := []string{"buildSubmitRequest", "parseSubmitResponse", "parseTaskResult"}
	if meta.FetchMode == "batch" {
		required = append(required, "buildBatchQueryRequest", "parseBatchResult")
	} else {
		required = append(required, "buildQueryRequest")
	}
	for _, hook := range required {
		ok, e := engine.HasCallablePath(context.Background(), hook)
		if e != nil {
			return nil, e
		}
		if !ok {
			return nil, fmt.Errorf("plugin %s is missing required export %q", meta.Key, hook)
		}
	}
	artifactHooks := map[string]bool{}
	for _, hook := range []string{"listArtifacts", "buildContentRequest"} {
		exported, e := engine.HasExport(context.Background(), hook)
		if e != nil {
			return nil, e
		}
		if !exported {
			continue
		}
		callable, e := engine.HasCallablePath(context.Background(), hook)
		if e != nil {
			return nil, e
		}
		if !callable {
			return nil, fmt.Errorf("plugin %s export %q is not a function", meta.Key, hook)
		}
		artifactHooks[hook] = true
	}
	if artifactHooks["listArtifacts"] != artifactHooks["buildContentRequest"] {
		return nil, fmt.Errorf("plugin %s must export listArtifacts and buildContentRequest together", meta.Key)
	}
	for _, route := range meta.Routes {
		for _, pair := range []struct{ kind, name string }{{"decode", route.Decode}, {"render", route.Render}} {
			if pair.name == "" {
				continue
			}
			ok, e := engine.HasCallablePath(context.Background(), "native", pair.name)
			if e != nil {
				return nil, e
			}
			if !ok {
				return nil, fmt.Errorf("plugin %s route %s %s references missing native %s %q", meta.Key, route.Method, route.Path, pair.kind, pair.name)
			}
		}
	}
	for _, claim := range meta.Protocols {
		def, known := HostProtocol(claim.Name)
		if !known {
			return nil, fmt.Errorf("plugin meta protocol %q is unknown", claim.Name)
		}
		requiredMembers := map[string]bool{}
		allowed := map[string]bool{}
		users := map[string][]string{}
		for _, op := range def.Operations {
			for _, h := range op.RequiredProtocolMembers {
				requiredMembers[h] = true
				allowed[h] = true
			}
			for _, m := range op.Modes {
				allowed[m.Hook] = true
				users[m.Hook] = append(users[m.Hook], m.Name)
				if slices.Contains(claim.Supports, m.Name) {
					requiredMembers[m.Hook] = true
				}
			}
		}
		for hook := range requiredMembers {
			ok, e := engine.HasCallablePath(context.Background(), "protocols", claim.Name, hook)
			if e != nil {
				return nil, e
			}
			if !ok {
				return nil, fmt.Errorf("plugin %s protocol %q is missing hook %q", meta.Key, claim.Name, hook)
			}
		}
		for hook := range allowed {
			if requiredMembers[hook] {
				continue
			}
			ok, e := engine.HasCallablePath(context.Background(), "protocols", claim.Name, hook)
			if e != nil {
				return nil, e
			}
			if ok {
				return nil, fmt.Errorf("plugin %s protocol %q exports unsupported hook %q", meta.Key, claim.Name, hook)
			}
		}
	}
	if protocols, e := engine.Export(context.Background(), "protocols"); e == nil {
		if object, ok := protocols.(map[string]any); ok {
			claimed := map[string]bool{}
			for _, claim := range meta.Protocols {
				claimed[claim.Name] = true
			}
			for name := range object {
				if !claimed[name] {
					return nil, fmt.Errorf("plugin %s implements unclaimed protocol %q", meta.Key, name)
				}
			}
		}
	}
	for _, removed := range []string{"resolveRequest", "renderError", "renderers"} {
		if ok, e := engine.HasExport(context.Background(), removed); e != nil {
			return nil, e
		} else if ok {
			return nil, fmt.Errorf("plugin %s export %q is no longer supported", meta.Key, removed)
		}
	}
	return &LoadedPlugin{Meta: meta, Engine: engine}, nil
}

func ValidateV1Meta(meta Meta) error {
	if meta.APIVersion != APIVersion1 {
		return fmt.Errorf("unsupported plugin apiVersion %d", meta.APIVersion)
	}
	meta.Key = strings.TrimSpace(meta.Key)
	if !pluginKeyPattern.MatchString(meta.Key) || len(meta.Key) > 30 {
		return fmt.Errorf("plugin meta key must match %s and not exceed 30 characters", pluginKeyPattern)
	}
	if strings.TrimSpace(meta.Name) == "" {
		return fmt.Errorf("plugin meta name is required")
	}
	if !pluginVersionPattern.MatchString(meta.Version) {
		return fmt.Errorf("plugin meta version must be semver")
	}
	if len(meta.Models) == 0 {
		return fmt.Errorf("plugin meta models must contain at least one model")
	}
	if meta.FetchMode != "per_task" && meta.FetchMode != "batch" {
		return fmt.Errorf("plugin meta fetchMode must be per_task or batch")
	}
	if err := validateLocalizedText(meta.Description, "description", 512); err != nil {
		return err
	}
	if strings.TrimSpace(meta.Author.Name) == "" {
		return fmt.Errorf("plugin meta author name is required")
	}
	if meta.Author.URL != "" {
		u, e := url.Parse(meta.Author.URL)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("plugin meta author url must be an absolute HTTP(S) URL")
		}
	}
	seen := map[string]bool{}
	for _, m := range meta.Models {
		if strings.TrimSpace(m) != m || m == "" || seen[m] {
			return fmt.Errorf("plugin meta models must be unique canonical names")
		}
		seen[m] = true
	}
	for _, c := range meta.Protocols {
		if _, ok := HostProtocol(c.Name); !ok {
			return fmt.Errorf("plugin meta protocol %q is unknown", c.Name)
		}
		if c.Name == "openai_responses" {
			if len(c.Supports) == 0 {
				return fmt.Errorf("plugin %s protocol %q must declare supports", meta.Key, c.Name)
			}
			definition, _ := HostProtocol(c.Name)
			modes := definition.DefinedModes()
			for _, s := range c.Supports {
				found := false
				for _, m := range modes {
					if m.Name == s {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("plugin %s protocol %q has no mode %q", meta.Key, c.Name, s)
				}
			}
		} else if len(c.Supports) > 0 {
			return fmt.Errorf("plugin %s protocol %q does not define modes; supports is not allowed", meta.Key, c.Name)
		}
	}
	for name, field := range meta.UsageSchema {
		if strings.TrimSpace(name) != name || name == "" {
			return fmt.Errorf("plugin meta usageSchema keys must be non-empty canonical names")
		}
		if err := validateUsageFieldSchema(name, field); err != nil {
			return err
		}
	}
	if len(meta.UsageExamples) > 0 && len(meta.UsageSchema) == 0 {
		return fmt.Errorf("plugin meta usageExamples requires usageSchema")
	}
	hasToken := false
	for _, field := range meta.UsageSchema {
		hasToken = hasToken || (field.Type == "number" && field.Unit == "token")
	}
	if hasToken && len(meta.UsageExamples) == 0 {
		return fmt.Errorf("plugin meta usageExamples is required when usageSchema declares a token unit")
	}
	for i, example := range meta.UsageExamples {
		label := strings.TrimSpace(example.Label)
		if label == "" {
			return fmt.Errorf("plugin meta usageExamples[%d] label is required", i)
		}
		if utf8.RuneCountInString(label) > 48 {
			return fmt.Errorf("plugin meta usageExamples[%d] label must not exceed 48 characters", i)
		}
		for key := range meta.UsageSchema {
			if _, ok := example.Facts[key]; !ok {
				return fmt.Errorf("plugin meta usageExamples[%d] facts missing key %q", i, key)
			}
		}
		for key, value := range example.Facts {
			field, ok := meta.UsageSchema[key]
			if !ok {
				return fmt.Errorf("plugin meta usageExamples[%d] facts has undeclared key %q", i, key)
			}
			if err := validateUsageExampleValue(value, field); err != nil {
				return fmt.Errorf("plugin meta usageExamples[%d] facts field %q %s", i, key, err)
			}
		}
	}
	return nil
}

func decodeMeta(value any) (Meta, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return Meta{}, fmt.Errorf("plugin meta must be an object")
	}
	for field := range object {
		switch field {
		case "apiVersion", "key", "name", "icon", "description", "version", "author", "channelTypes", "models", "fetchMode", "allowedHosts", "routes", "protocols", "usageSchema", "usageExamples", "auth":
		default:
			return Meta{}, fmt.Errorf("plugin meta has unknown field %q", field)
		}
	}
	if auth, exists := object["auth"]; exists {
		if text, ok := auth.(string); ok {
			object["auth"] = map[string]any{"type": text}
		}
	}
	encoded, err := common.Marshal(object)
	if err != nil {
		return Meta{}, fmt.Errorf("encode plugin meta: %w", err)
	}
	var m Meta
	if err = common.Unmarshal(encoded, &m); err != nil {
		return Meta{}, fmt.Errorf("decode plugin meta: %w", err)
	}
	return m, nil
}

func validateUsageExampleValue(value any, field UsageFieldSchema) error {
	if field.Enum != nil {
		text, ok := value.(string)
		if !ok || !slices.Contains(field.Enum, text) {
			return fmt.Errorf("enum is not an allowed value")
		}
		return nil
	}
	if field.Type == "boolean" {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}
		return nil
	}
	number, ok := value.(float64)
	if !ok {
		if n, yes := value.(int64); yes {
			number, ok = float64(n), true
		}
	}
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return fmt.Errorf("must be a finite non-negative number")
	}
	limit := 2147483647.0
	if field.Unit == "second" {
		limit = 3600
	}
	if field.Unit == "count" {
		limit = 128
	}
	if number > limit {
		return fmt.Errorf("exceeds the host limit")
	}
	return nil
}

func validateLocalizedText(text LocalizedText, name string, max int) error {
	if text == nil {
		return nil
	}
	if len(text) > 16 {
		return fmt.Errorf("plugin meta %s must not exceed 16 locales", name)
	}
	for locale, v := range text {
		if !localeTagPattern.MatchString(locale) {
			return fmt.Errorf("plugin meta %s has invalid locale %q", name, locale)
		}
		v = strings.TrimSpace(v)
		if v == "" {
			return fmt.Errorf("plugin meta %s value must be non-empty", name)
		}
		for _, r := range v {
			if unicode.IsControl(r) {
				return fmt.Errorf("plugin meta %s must not contain control characters", name)
			}
		}
		if utf8.RuneCountInString(v) > max {
			return fmt.Errorf("plugin meta %s exceeds %d characters", name, max)
		}
	}
	if _, ok := text["en"]; !ok {
		return fmt.Errorf("plugin meta %s must include a non-empty \"en\" value", name)
	}
	return nil
}
func validateUsageFieldSchema(name string, f UsageFieldSchema) error {
	if f.Enum != nil {
		if f.Type != "" || f.Unit != "" || len(f.Enum) == 0 {
			return fmt.Errorf("plugin meta usageSchema field %q enum is invalid", name)
		}
		return nil
	}
	if f.Type == "boolean" {
		if f.Unit != "" {
			return fmt.Errorf("plugin meta usageSchema field %q cannot combine boolean with unit", name)
		}
		return nil
	}
	if f.Type != "number" || !slices.Contains([]string{"second", "count", "token", "credit"}, f.Unit) {
		return fmt.Errorf("plugin meta usageSchema field %q type or unit is invalid", name)
	}
	return nil
}
