// Package apple implements an independent, ephemeral client for the private Photos service.
package apple

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/jaysonhurd/icloud-dl/internal/archive"
)

const (
	authRoot  = "https://idmsa.apple.com/appleauth/auth"
	setupRoot = "https://setup.icloud.com/setup/ws/1/accountLogin"
	widget    = "d39ba9916b7251055b22c7f910e2ea796ee65e98b2ddecea8f5dde8d9d1a815d"
)

var (
	// ErrAuth indicates failed/unsupported Apple authentication without exposing response data.
	ErrAuth = errors.New("Apple authentication failed or requires an unsupported challenge")
	// ErrCatalog indicates an unavailable, indexing, or changed private Photos schema.
	ErrCatalog = errors.New("Photos catalog unavailable or incomplete; service schema or account access may have changed")
)

// Prompt requests terminal input or consent, never through configuration or logs.
type Prompt func(context.Context, string, bool) (string, error)

// Source holds only ephemeral authentication state; no password or media URLs are stored.
type Source struct {
	lastTokenIssued bool
	trustToken      string
	client          ClientDoer
	headers         http.Header
	service         string
	owners          map[string]string
}

// New constructs a nonredirecting cookie client isolated from the media client.
func New() *Source {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 30 * time.Second, ResponseHeaderTimeout: 60 * time.Second, IdleConnTimeout: 90 * time.Second}
	return &Source{client: &http.Client{Jar: jar, Transport: transport, Timeout: 90 * time.Second, CheckRedirect: RedirectPolicy}, headers: make(http.Header), owners: map[string]string{}}
}

// allowedAPI admits fixed auth/setup authorities and the one discovered Apple service.
func (s *Source) allowedAPI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	if u.Host == "idmsa.apple.com" || u.Host == "setup.icloud.com" {
		return true
	}
	service, err := url.Parse(s.service)
	return err == nil && service.Host != "" && u.Host == service.Host
}

// request applies bounded JSON decoding and returns only safe categories and status codes.
func (s *Source) request(ctx context.Context, method, endpoint string, payload, out any) (int, error) {
	if !s.allowedAPI(endpoint) {
		return 0, ErrEndpoint
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, ErrHTTP
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return 0, ErrHTTP
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15 Safari/605.1.15")
	req.Header.Set("Origin", "https://www.icloud.com")
	req.Header.Set("Referer", "https://www.icloud.com/")
	if strings.HasPrefix(endpoint, authRoot) {
		req.Header.Set("Origin", "https://idmsa.apple.com")
		req.Header.Set("Referer", "https://idmsa.apple.com/")
		for k, v := range s.headers {
			req.Header[k] = append([]string(nil), v...)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, ErrHTTP
	}
	defer resp.Body.Close()
	s.lastTokenIssued = resp.Header.Get("X-Apple-Session-Token") != ""
	if strings.HasPrefix(endpoint, authRoot) {
		if token := resp.Header.Get("X-Apple-TwoSV-Trust-Token"); token != "" {
			s.trustToken = token
		}
		for _, name := range []string{"scnt", "X-Apple-ID-Session-Id", "X-Apple-Auth-Attributes", "X-Apple-Session-Token", "X-Apple-ID-Account-Country"} {
			if v := resp.Header.Get(name); v != "" {
				s.headers.Set(name, v)
			}
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (16<<20)+1))
	if err != nil || len(data) > 16<<20 {
		return resp.StatusCode, ErrHTTP
	}
	if out != nil && resp.StatusCode == 200 {
		if json.Unmarshal(data, out) != nil {
			return resp.StatusCode, ErrHTTP
		}
	}
	return resp.StatusCode, nil
}

// Login performs normal SRP verification; device delivery and trust require explicit consent.
// It stores no session files and deliberately rejects repair/security-setting flows.
func (s *Source) Login(ctx context.Context, account, password string, prompt Prompt, allowTrust bool) error {
	s.trustToken = ""
	trustGranted := false
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return ErrAuth
	}
	frame := "auth-" + hex.EncodeToString(random)
	for k, v := range map[string]string{"X-Apple-Widget-Key": widget, "X-Apple-OAuth-Client-Id": widget, "X-Apple-OAuth-Client-Type": "firstPartyAuth", "X-Apple-OAuth-Redirect-URI": "https://www.icloud.com", "X-Apple-OAuth-Require-Grant-Code": "true", "X-Apple-OAuth-Response-Mode": "web_message", "X-Apple-OAuth-Response-Type": "code", "X-Apple-OAuth-State": frame, "X-Apple-Frame-Id": frame} {
		s.headers.Set(k, v)
	}
	params := url.Values{"frame_id": {frame}, "iframeId": {frame}, "client_id": {widget}, "redirect_uri": {"https://www.icloud.com"}, "response_type": {"code"}, "response_mode": {"web_message"}, "state": {frame}, "authVersion": {"latest"}, "language": {"en_US"}, "skVersion": {"7"}}
	status, err := s.request(ctx, "GET", authRoot+"/authorize/signin?"+params.Encode(), nil, nil)
	if err != nil || status != 200 {
		return fmt.Errorf("Apple sign-in initialization failed (HTTP %d): %w", status, ErrAuth)
	}
	status, err = s.request(ctx, "POST", authRoot+"/federate", map[string]any{"accountName": account, "rememberMe": false}, nil)
	if err != nil || status != 200 {
		return fmt.Errorf("Apple account federation failed (HTTP %d): %w", status, ErrAuth)
	}
	secret, public, err := srpStart()
	if err != nil {
		return err
	}
	defer clear(secret)
	var challenge struct {
		Iteration int    `json:"iteration"`
		Salt      string `json:"salt"`
		Protocol  string `json:"protocol"`
		B         string `json:"b"`
		C         string `json:"c"`
	}
	status, err = s.request(ctx, "POST", authRoot+"/signin/init", map[string]any{"a": public, "accountName": account, "protocols": []string{"s2k", "s2k_fo"}}, &challenge)
	if err != nil || status != 200 {
		return ErrAuth
	}
	salt, e1 := base64.StdEncoding.DecodeString(challenge.Salt)
	server, e2 := base64.StdEncoding.DecodeString(challenge.B)
	if e1 != nil || e2 != nil {
		return ErrAuth
	}
	m1, m2, err := srpProof(secret, account, password, salt, server, challenge.Iteration, challenge.Protocol)
	if err != nil {
		return err
	}
	status, err = s.request(ctx, "POST", authRoot+"/signin/complete", map[string]any{"accountName": account, "m1": m1, "m2": m2, "c": challenge.C, "rememberMe": false, "trustTokens": []string{}}, nil)
	if err != nil {
		return err
	}
	if status == 409 {
		consent, err := prompt(ctx, "Send an Apple verification prompt to your trusted device? [y/N]: ", false)
		if err != nil {
			return err
		}
		if !strings.EqualFold(consent, "y") {
			return ErrAuth
		}
		status, err = s.request(ctx, "PUT", authRoot+"/verify/trusteddevice/securitycode", nil, nil)
		if err != nil {
			return fmt.Errorf("Apple verification notification request failed: %w", ErrAuth)
		}
		// Delivery is not authentication. An accepted request or an outstanding
		// challenge may still require the user to enter the code on their device.
		// Always validate that code separately before account login or trust.
		switch status {
		case http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusConflict:
		default:
			return fmt.Errorf("Apple verification notification request returned HTTP %d: %w", status, ErrAuth)
		}
		code, err := prompt(ctx, "Apple verification code: ", true)
		if err != nil {
			return err
		}
		if len(code) != 6 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return ErrAuth
		}
		status, err = s.request(ctx, "POST", authRoot+"/verify/trusteddevice/securitycode", map[string]any{"securityCode": map[string]string{"code": code}}, nil)
		if err != nil || (status != 200 && status != 204 && !(status == 409 && s.lastTokenIssued)) {
			return fmt.Errorf("Apple verification code was not accepted (HTTP %d): %w", status, ErrAuth)
		}
		if allowTrust {
			consent, err = prompt(ctx, "Allow Apple's session trust operation for this login? No credentials will be saved locally. [y/N]: ", false)
			if err != nil {
				return err
			}
			if strings.EqualFold(consent, "y") {
				status, err = s.request(ctx, "GET", authRoot+"/2sv/trust", nil, nil)
				if err != nil || (status != http.StatusOK && status != http.StatusNoContent) {
					return fmt.Errorf("Apple session trust failed (HTTP %d): %w", status, ErrAuth)
				}
				trustGranted = true
			}
		}
	} else if status != 200 {
		return ErrAuth
	}
	token := s.headers.Get("X-Apple-Session-Token")
	if token == "" {
		return fmt.Errorf("Apple did not issue a session token after verification: %w", ErrAuth)
	}
	var accountInfo struct {
		Webservices map[string]struct {
			URL    string `json:"url"`
			Status string `json:"status"`
		} `json:"webservices"`
	}
	loginPayload := map[string]any{"accountCountryCode": s.headers.Get("X-Apple-ID-Account-Country"), "dsWebAuthToken": token, "extended_login": false}
	if trustGranted && s.trustToken != "" {
		loginPayload["trustToken"] = s.trustToken
	}
	status, err = s.request(ctx, "POST", setupRoot, loginPayload, &accountInfo)
	if err != nil {
		return fmt.Errorf("Apple account-login response could not be processed (HTTP %d): %w", status, ErrAuth)
	}
	if status != http.StatusOK {
		return fmt.Errorf("Apple account login failed after verification (HTTP %d): %w", status, ErrAuth)
	}
	endpoint, present := accountInfo.Webservices["ckdatabasews"]
	service, err := photosEndpoint(endpoint.URL, endpoint.Status, present)
	if err != nil {
		return err
	}
	// Only this authenticated, validated service authority becomes eligible for requests.
	s.service = service
	return nil
}

// record retains only protocol fields needed for asset/master association.
type record struct {
	Name   string `json:"recordName"`
	Type   string `json:"recordType"`
	Fields map[string]struct {
		Value jsontext.Value `json:"value"`
		Type  string         `json:"type"`
	} `json:"fields"`
}

// query sends a read-only CloudKit operation in the primary personal zone.
func (s *Source) query(ctx context.Context, operation string, payload map[string]any) ([]record, error) {
	if s.service == "" {
		return nil, ErrAuth
	}
	payload["zoneID"] = map[string]string{"zoneName": "PrimarySync"}
	var response struct {
		Records []record `json:"records"`
	}
	status, err := s.request(ctx, "POST", s.service+operation, payload, &response)
	if err != nil || status != 200 || response.Records == nil {
		return nil, ErrCatalog
	}
	return response.Records, nil
}

// List enumerates assets by rank and associates masters across all received pages.
func (s *Source) List(ctx context.Context) ([]archive.Asset, error) {
	ready, err := s.query(ctx, "records/query", map[string]any{"query": map[string]string{"recordType": "CheckIndexingState"}})
	if err != nil || len(ready) == 0 {
		return nil, ErrCatalog
	}
	var indexing string
	_ = json.Unmarshal(ready[0].Fields["state"].Value, &indexing)
	if indexing != "FINISHED" {
		return nil, ErrCatalog
	}
	masters := map[string]record{}
	assets := map[string]record{}
	order := []string{}
	rank := 0
	var listingErr error
	for {
		page, err := s.query(ctx, "records/query", map[string]any{"query": map[string]any{"recordType": "CPLAssetAndMasterByAssetDateWithoutHiddenOrDeleted", "filterBy": []any{
			map[string]any{"fieldName": "startRank", "comparator": "EQUALS", "fieldValue": map[string]any{"type": "INT64", "value": rank}},
			map[string]any{"fieldName": "direction", "comparator": "EQUALS", "fieldValue": map[string]string{"type": "STRING", "value": "ASCENDING"}},
		}}, "resultsLimit": 200})
		if err != nil {
			listingErr = err
			break
		}
		count, newAssets := 0, 0
		for _, r := range page {
			if r.Name == "" {
				listingErr = ErrCatalog
				continue
			}
			switch r.Type {
			case "CPLMaster":
				masters[r.Name] = r
			case "CPLAsset":
				count++
				if _, exists := assets[r.Name]; !exists {
					order = append(order, r.Name)
					newAssets++
				}
				assets[r.Name] = r
			default:
				listingErr = ErrCatalog
			}
		}
		if count == 0 {
			if len(page) > 0 {
				listingErr = ErrCatalog
			}
			break
		}
		if newAssets == 0 {
			listingErr = ErrCatalog
			break
		}
		rank += count
		if rank > 1000000 {
			listingErr = ErrCatalog
			break
		}
	}
	result := make([]archive.Asset, 0, len(order))
	for _, id := range order {
		asset := archive.Asset{ID: id, Size: -1}
		var ref struct {
			Name string `json:"recordName"`
		}
		_ = json.Unmarshal(assets[id].Fields["masterRef"].Value, &ref)
		master, exists := masters[ref.Name]
		if exists {
			var name string
			_ = json.Unmarshal(master.Fields["filenameEnc"].Value, &name)
			if master.Fields["filenameEnc"].Type != "STRING" {
				if b, err := base64.StdEncoding.DecodeString(name); err == nil {
					name = string(b)
				} else {
					name = ""
				}
			}
			asset.Name = name
			var resource struct {
				Size *int64 `json:"size"`
			}
			err := json.Unmarshal(master.Fields["resOriginalRes"].Value, &resource)
			_, compound := master.Fields["resOriginalVidComplRes"]
			asset.Supported = err == nil && name != "" && !compound
			if resource.Size != nil {
				asset.Size = *resource.Size
			}
			s.owners[id] = ref.Name
		}
		result = append(result, asset)
	}
	return result, listingErr
}

// URL resolves the original from its master immediately before the transfer.
func (s *Source) URL(ctx context.Context, asset archive.Asset) (string, error) {
	owner := s.owners[asset.ID]
	if owner == "" {
		return "", ErrCatalog
	}
	records, err := s.query(ctx, "records/lookup", map[string]any{"records": []map[string]string{{"recordName": owner}}})
	if err != nil || len(records) != 1 || records[0].Name != owner {
		return "", ErrCatalog
	}
	var resource struct {
		URL string `json:"downloadURL"`
	}
	if json.Unmarshal(records[0].Fields["resOriginalRes"].Value, &resource) != nil {
		return "", ErrCatalog
	}
	if !mediaAuthority(resource.URL) {
		return "", ErrEndpoint
	}
	return resource.URL, nil
}

// mediaAuthority admits only HTTPS Apple content hosts, never a service URL or userinfo.
func mediaAuthority(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.User == nil && u.Port() == "" && strings.HasSuffix(u.Hostname(), ".icloud-content.com")
}

// MediaClient is a credential-free downloader with redirects disabled.
type MediaClient struct{ client *http.Client }

// NewMediaClient creates an independent streaming transport; no account cookie jar is shared.
func NewMediaClient() *MediaClient {
	return &MediaClient{client: &http.Client{Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 30 * time.Second, ResponseHeaderTimeout: 60 * time.Second}, CheckRedirect: RedirectPolicy}}
}

// Do rejects unapproved media authorities before any request is sent.
func (m *MediaClient) Do(req *http.Request) (*http.Response, error) {
	if !mediaAuthority(req.URL.String()) || req.Method != "GET" || req.Body != nil {
		return nil, ErrEndpoint
	}
	clean := req.Clone(req.Context())
	clean.Header = make(http.Header)
	resp, err := m.client.Do(clean)
	if err != nil {
		return nil, ErrHTTP
	}
	return resp, nil
}

// Summary prints only allowlisted outcome counts, never record names or URLs.
func Summary(reportItems int, limited bool) string {
	return fmt.Sprintf("Observed outcomes: %d; limited selection: %t", reportItems, limited)
}

// photosEndpoint validates discovery using fixed diagnostic reasons, never the raw URL.
// Status is optional in the observed icloudpd lookup; an explicit inactive value still fails.
func photosEndpoint(raw, status string, present bool) (string, error) {
	reject := func(reason string) (string, error) {
		return "", fmt.Errorf("Apple Photos service discovery: %s: %w", reason, ErrEndpoint)
	}
	if !present {
		return reject("account login did not return ckdatabasews; this session may require additional account approval")
	}
	if raw == "" {
		return reject("ckdatabasews has no service URL")
	}
	if status != "" && status != "active" {
		return reject("ckdatabasews reported an inactive or unrecognized service status")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return reject("service URL could not be parsed")
	}
	if u.Scheme != "https" {
		return reject("service URL does not use HTTPS")
	}
	if u.User != nil {
		return reject("service URL contains disallowed user information")
	}
	if u.Port() != "" && u.Port() != "443" {
		return reject("service URL uses an unapproved port")
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, "-ckdatabasews.icloud.com") {
		if strings.HasSuffix(host, ".icloud.com") {
			return reject("service host is under icloud.com but does not match the supported Photos authority pattern")
		}
		return reject("service host is outside the supported global Photos authority pattern; regional services are not implemented")
	}
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return reject("service URL has unexpected query or fragment data")
	}
	if u.Path != "" && u.Path != "/" {
		return reject("service URL has an unexpected path")
	}
	// Normalize explicit default ports before applying the service HTTP boundary.
	return "https://" + host + "/database/1/com.apple.photos.cloud/production/private/", nil
}
