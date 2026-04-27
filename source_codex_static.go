package modeldb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const codexSourceID = "codex-api"

const (
	defaultCodexModelsEndpoint = "https://chatgpt.com/backend-api/codex/models"
	defaultCodexClientVersion  = "0.124.0"
	codexTokenEndpoint         = "https://auth.openai.com/oauth/token"
	codexClientID              = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexAuthFilePath          = ".codex/auth.json"
	codexTokenExpiryBuffer     = 5 * time.Minute
)

const (
	envCodexAuthPath    = "CODEX_AUTH_PATH"
	envCodexAccessToken = "CODEX_ACCESS_TOKEN"
	envCodexOAuthToken  = "CODEX_CODE_OAUTH_TOKEN"
)

type CodexSource struct {
	FilePath      string
	AccessToken   string
	AuthPath      string
	ModelsURL     string
	ClientVersion string
	Client        *http.Client
}

func NewCodexSource() CodexSource {
	return CodexSource{
		AccessToken:   firstNonEmpty(os.Getenv(envCodexAccessToken), os.Getenv(envCodexOAuthToken)),
		AuthPath:      os.Getenv(envCodexAuthPath),
		ModelsURL:     defaultCodexModelsEndpoint,
		ClientVersion: defaultCodexClientVersion,
		Client:        http.DefaultClient,
	}
}
func NewCodexSourceFromFile(path string) CodexSource { return CodexSource{FilePath: path} }
func DefaultCodexFixturePath() string {
	return filepath.Join("internal", "source", "codex", "testdata", "models.json")
}
func (CodexSource) ID() string { return codexSourceID }

type codexReasoningLevel struct {
	Effort string `json:"effort"`
}
type codexModelEntry struct {
	Slug                     string                `json:"slug"`
	DisplayName              string                `json:"display_name"`
	Description              string                `json:"description"`
	DefaultReasoningLevel    string                `json:"default_reasoning_level"`
	SupportedReasoningLevels []codexReasoningLevel `json:"supported_reasoning_levels"`
	SupportedInAPI           bool                  `json:"supported_in_api"`
	SupportVerbosity         bool                  `json:"support_verbosity"`
	DefaultVerbosity         string                `json:"default_verbosity"`
	SupportsReasoningSummary bool                  `json:"supports_reasoning_summaries"`
	DefaultReasoningSummary  string                `json:"default_reasoning_summary"`
	ContextWindow            int                   `json:"context_window"`
	InputModalities          []string              `json:"input_modalities"`
	OutputModalities         []string              `json:"output_modalities"`
	LastUpdated              string                `json:"last_updated"`
	Deprecated               bool                  `json:"deprecated"`
	SupportsParallelTools    bool                  `json:"supports_parallel_tool_calls"`
}

type codexPayload struct {
	Models []codexModelEntry `json:"models"`
}

func (s CodexSource) Fetch(ctx context.Context) (*Fragment, error) {
	payload, err := s.loadPayload(ctx)
	if err != nil {
		return nil, err
	}
	return fragmentFromCodexPayload(payload)
}

func (s CodexSource) loadPayload(ctx context.Context) (codexPayload, error) {
	if s.FilePath == "" {
		return s.fetchLivePayload(ctx)
	}
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return codexPayload{}, err
	}
	var payload codexPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return codexPayload{}, err
	}
	return payload, nil
}

func (s CodexSource) fetchLivePayload(ctx context.Context) (codexPayload, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	auth, err := loadCodexSourceAuth(s, client)
	if err != nil {
		return codexPayload{}, err
	}
	token, err := auth.token(ctx)
	if err != nil {
		return codexPayload{}, err
	}
	modelsURL := s.ModelsURL
	if modelsURL == "" {
		modelsURL = defaultCodexModelsEndpoint
	}
	reqURL, err := codexModelsURL(modelsURL, firstNonEmpty(s.ClientVersion, defaultCodexClientVersion))
	if err != nil {
		return codexPayload{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return codexPayload{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if auth.accountID != "" {
		req.Header.Set("ChatGPT-Account-ID", auth.accountID)
	}
	req.Header.Set("originator", "codex_cli_rs")
	req.Header.Set("version", firstNonEmpty(s.ClientVersion, defaultCodexClientVersion))

	resp, err := client.Do(req)
	if err != nil {
		return codexPayload{}, fmt.Errorf("codex source: list models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return codexPayload{}, fmt.Errorf("codex source: HTTP %d: %s", resp.StatusCode, string(body))
	}
	var payload codexPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return codexPayload{}, fmt.Errorf("codex source: decode models response: %w", err)
	}
	return payload, nil
}

func fragmentFromCodexPayload(payload codexPayload) (*Fragment, error) {
	observedAt := time.Time{}
	frag := &Fragment{Services: []Service{{ID: "codex", Name: "Codex", Kind: ServiceKindDirect, Operator: "openai", DocsURL: "https://chatgpt.com/codex", Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt}}}}}
	for _, item := range payload.Models {
		if !item.SupportedInAPI {
			continue
		}
		key, ok := inferOpenAIModelKey(item.Slug)
		if !ok {
			continue
		}
		modelCaps := coarseCachingCapabilities(capabilitiesFromCodexModel(item), true)
		exposureCaps := capabilitiesFromCodexModel(item)
		frag.Models = append(frag.Models, ModelRecord{Key: key, Name: item.DisplayName, Description: item.Description, Canonical: false, Capabilities: modelCaps, Limits: Limits{ContextWindow: item.ContextWindow}, InputModalities: normalizeStrings(item.InputModalities), OutputModalities: normalizeStrings(item.OutputModalities), LastUpdated: normalizeDate(item.LastUpdated), Deprecated: item.Deprecated, Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}}})
		exp := OfferingExposure{APIType: APITypeOpenAIResponses, ExposedCapabilities: capabilitiesPtr(exposureCaps), SupportedParameters: codexSupportedParameters(item), ParameterMappings: codexParameterMappings(item), ParameterValues: codexParameterValues(item), Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}}}
		frag.Offerings = append(frag.Offerings, Offering{ServiceID: "codex", WireModelID: item.Slug, ModelKey: key, Exposures: []OfferingExposure{exp}, Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}}})
	}
	sort.Slice(frag.Offerings, func(i, j int) bool { return frag.Offerings[i].WireModelID < frag.Offerings[j].WireModelID })
	return frag, nil
}

func codexModelsURL(rawURL, clientVersion string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if q.Get("client_version") == "" {
		q.Set("client_version", clientVersion)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type codexSourceAuthFile struct {
	AuthMode string `json:"auth_mode"`
	Tokens   struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		AccountID    string `json:"account_id"`
	} `json:"tokens"`
	LastRefresh time.Time `json:"last_refresh"`
}

type codexSourceAuth struct {
	accessToken  string
	refreshToken string
	accountID    string
	path         string
	expiry       time.Time
	client       *http.Client
	file         codexSourceAuthFile
}

func loadCodexSourceAuth(source CodexSource, client *http.Client) (*codexSourceAuth, error) {
	if source.AccessToken != "" {
		return &codexSourceAuth{accessToken: source.AccessToken, client: client}, nil
	}
	path := source.AuthPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("codex source: get home dir: %w", err)
		}
		path = filepath.Join(home, codexAuthFilePath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("codex source: read %s: %w", path, err)
	}
	var file codexSourceAuthFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("codex source: parse auth file: %w", err)
	}
	if file.AuthMode != "" && file.AuthMode != "chatgpt" {
		return nil, fmt.Errorf("codex source: unsupported auth mode %q", file.AuthMode)
	}
	if file.Tokens.AccessToken == "" && file.Tokens.RefreshToken == "" {
		return nil, fmt.Errorf("codex source: no tokens in %s", path)
	}
	auth := &codexSourceAuth{
		accessToken:  file.Tokens.AccessToken,
		refreshToken: file.Tokens.RefreshToken,
		accountID:    file.Tokens.AccountID,
		path:         path,
		client:       client,
		file:         file,
	}
	if exp, err := jwtExpiry(file.Tokens.AccessToken); err == nil {
		auth.expiry = exp
	}
	return auth, nil
}

func (a *codexSourceAuth) token(ctx context.Context) (string, error) {
	if a.expiry.IsZero() && a.accessToken != "" && a.refreshToken == "" {
		return a.accessToken, nil
	}
	if !a.expiry.IsZero() && time.Now().Add(codexTokenExpiryBuffer).Before(a.expiry) {
		return a.accessToken, nil
	}
	if a.refreshToken == "" {
		if a.accessToken != "" {
			return a.accessToken, nil
		}
		return "", fmt.Errorf("codex source: no access token")
	}
	return a.refresh(ctx)
}

func (a *codexSourceAuth) refresh(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {a.refreshToken},
		"client_id":     {codexClientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("codex source: build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := a.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("codex source: token refresh: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("codex source: decode refresh response (status %d): %w", resp.StatusCode, err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("codex source: token refresh failed: %s: %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("codex source: empty access token in refresh response (status %d)", resp.StatusCode)
	}
	a.accessToken = result.AccessToken
	a.file.Tokens.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		a.refreshToken = result.RefreshToken
		a.file.Tokens.RefreshToken = result.RefreshToken
	}
	if result.ExpiresIn > 0 {
		a.expiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	} else if exp, err := jwtExpiry(result.AccessToken); err == nil {
		a.expiry = exp
	} else {
		a.expiry = time.Time{}
	}
	if a.path != "" {
		a.file.LastRefresh = time.Now().UTC()
		if data, err := json.MarshalIndent(a.file, "", "  "); err == nil {
			_ = os.WriteFile(a.path, data, 0o600)
		}
	}
	return a.accessToken, nil
}

func jwtExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("decode JWT payload: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.NewDecoder(bytes.NewReader(payload)).Decode(&claims); err != nil {
		return time.Time{}, fmt.Errorf("decode JWT claims: %w", err)
	}
	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("JWT has no exp claim")
	}
	return time.Unix(claims.Exp, 0), nil
}

func coarseCachingCapabilities(caps Capabilities, available bool) Capabilities {
	if !available {
		return caps
	}
	caps.Caching = &CachingCapability{Available: true}
	return caps
}

func capabilitiesFromCodexModel(item codexModelEntry) Capabilities {
	caps := Capabilities{ToolUse: true, ParallelToolCalls: item.SupportsParallelTools, StructuredOutput: true, Streaming: true, Temperature: true, Vision: containsString(item.InputModalities, "image"), Caching: &CachingCapability{Available: true, Mode: CachingModeImplicit}}
	efforts := make([]ReasoningEffortLevel, 0, len(item.SupportedReasoningLevels)+1)
	if !strings.Contains(strings.ToLower(item.Slug), "mini") {
		efforts = append(efforts, ReasoningEffortNone)
	}
	for _, e := range item.SupportedReasoningLevels {
		switch strings.ToLower(strings.TrimSpace(e.Effort)) {
		case "low":
			efforts = append(efforts, ReasoningEffortLow)
		case "medium":
			efforts = append(efforts, ReasoningEffortMedium)
		case "high":
			efforts = append(efforts, ReasoningEffortHigh)
		case "max":
			efforts = append(efforts, ReasoningEffortMax)
		case "xhigh":
			efforts = append(efforts, ReasoningEffortXHigh)
		}
	}
	if len(efforts) > 0 || item.SupportsReasoningSummary {
		caps.Reasoning = &ReasoningCapability{Available: true, Efforts: dedupeEfforts(efforts), Summaries: codexSummaryValues(item.SupportsReasoningSummary), Modes: []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff}, VisibleSummary: item.SupportsReasoningSummary}
	}
	return caps
}

func codexSummaryValues(enabled bool) []ReasoningSummaryValue {
	if !enabled {
		return nil
	}
	return []ReasoningSummaryValue{ReasoningSummaryNone, ReasoningSummaryAuto, ReasoningSummaryConcise, ReasoningSummaryDetailed}
}

func dedupeEfforts(in []ReasoningEffortLevel) []ReasoningEffortLevel {
	seen := map[ReasoningEffortLevel]bool{}
	out := make([]ReasoningEffortLevel, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func codexSupportedParameters(item codexModelEntry) []NormalizedParameter {
	params := []NormalizedParameter{ParamResponseFormat, ParamTools, ParamTemperature}
	if item.SupportsParallelTools {
		params = append(params, ParamParallelTools)
	}
	if len(item.SupportedReasoningLevels) > 0 {
		params = append(params, ParamThinking, ParamReasoningEffort)
	}
	if item.SupportsReasoningSummary {
		params = append(params, ParamReasoningSummary)
	}
	return normalizeNormalizedParameters(params)
}

func codexParameterMappings(item codexModelEntry) []ParameterMapping {
	m := []ParameterMapping{{Normalized: ParamResponseFormat, WireName: "response_format"}, {Normalized: ParamTools, WireName: "tools"}, {Normalized: ParamTemperature, WireName: "temperature"}}
	if item.SupportsParallelTools {
		m = append(m, ParameterMapping{Normalized: ParamParallelTools, WireName: "parallel_tool_calls"})
	}
	if len(item.SupportedReasoningLevels) > 0 {
		m = append(m, ParameterMapping{Normalized: ParamThinking, WireName: "reasoning"}, ParameterMapping{Normalized: ParamReasoningEffort, WireName: "reasoning.effort"})
	}
	if item.SupportsReasoningSummary {
		m = append(m, ParameterMapping{Normalized: ParamReasoningSummary, WireName: "reasoning.summary"})
	}
	return m
}

func codexParameterValues(item codexModelEntry) map[string][]string {
	values := map[string][]string{}
	efforts := make([]string, 0, len(item.SupportedReasoningLevels)+1)
	if !strings.Contains(strings.ToLower(item.Slug), "mini") {
		efforts = append(efforts, string(ReasoningEffortNone))
	}
	for _, e := range item.SupportedReasoningLevels {
		s := strings.ToLower(strings.TrimSpace(e.Effort))
		switch s {
		case "low", "medium", "high", "max", "xhigh":
			efforts = append(efforts, s)
		}
	}
	if len(efforts) > 0 {
		values[string(ParamReasoningEffort)] = efforts
	}
	if item.SupportsReasoningSummary {
		values[string(ParamReasoningSummary)] = []string{string(ReasoningSummaryAuto), string(ReasoningSummaryConcise), string(ReasoningSummaryDetailed)}
	}
	if item.SupportVerbosity {
		values["verbosity"] = []string{"low", "medium", "high"}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
