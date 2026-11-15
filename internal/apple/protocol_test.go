package apple

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/jaysonhurd/icloud-dl/internal/archive"
)

// TestSRPAgreement constructs a synthetic server independently to check shared-secret agreement.
func TestSRPAgreement(t *testing.T) {
	n, _ := new(big.Int).SetString(groupHex, 16)
	g := big.NewInt(2)
	pad := func(v *big.Int) []byte { return v.FillBytes(make([]byte, 256)) }
	secret := make([]byte, 32)
	secret[31] = 7
	salt := []byte("synthetic-salt")
	keyHash := sha256.Sum256([]byte("synthetic-password"))
	derived, err := pbkdf2.Key(sha256.New, string(keyHash[:]), salt, 100, 32)
	if err != nil {
		t.Fatal(err)
	}
	inner := sha256.Sum256(append([]byte(":"), derived...))
	xh := sha256.Sum256(append(salt, inner[:]...))
	x := new(big.Int).SetBytes(xh[:])
	v := new(big.Int).Exp(g, x, n)
	b := big.NewInt(11)
	kh := sha256.Sum256(append(pad(n), pad(g)...))
	k := new(big.Int).SetBytes(kh[:])
	B := new(big.Int).Mod(new(big.Int).Add(new(big.Int).Mul(k, v), new(big.Int).Exp(g, b, n)), n)
	A := new(big.Int).Exp(g, big.NewInt(7), n)
	uh := sha256.Sum256(append(pad(A), pad(B)...))
	u := new(big.Int).SetBytes(uh[:])
	shared := new(big.Int).Exp(new(big.Int).Mul(A, new(big.Int).Exp(v, u, n)), b, n)
	sessionKey := sha256.Sum256(pad(shared))
	hn := sha256.Sum256(pad(n))
	hg := sha256.Sum256(pad(g))
	for i := range hn {
		hn[i] ^= hg[i]
	}
	hi := sha256.Sum256([]byte("synthetic-account"))
	h := sha256.New()
	for _, part := range [][]byte{hn[:], hi[:], salt, pad(A), pad(B), sessionKey[:]} {
		h.Write(part)
	}
	want := base64.StdEncoding.EncodeToString(h.Sum(nil))
	actual, _, err := srpProof(secret, "synthetic-account", "synthetic-password", salt, pad(B), 100, "s2k")
	if err != nil || actual != want {
		t.Fatalf("SRP agreement failed: %v", err)
	}
	for _, bad := range [][]byte{nil, make([]byte, 256), pad(n)} {
		if _, _, err := srpProof(secret, "synthetic-account", "synthetic-password", salt, bad, 100, "s2k"); err == nil {
			t.Fatal("accepted unsafe server value")
		}
	}
}

// TestLoginSynthetic exercises login/discovery without contacting Apple or persisting secrets.
func TestLoginSynthetic(t *testing.T) {
	s := New()
	calls := 0
	s.client = doer(func(req *http.Request) (*http.Response, error) {
		calls++
		body := "{}"
		headers := make(http.Header)
		switch {
		case strings.HasSuffix(req.URL.Path, "/signin/init"):
			body = `{"iteration":100,"salt":"c3ludGhldGlj","protocol":"s2k","b":"Ag==","c":"synthetic"}`
		case strings.HasSuffix(req.URL.Path, "/signin/complete"):
			headers.Set("X-Apple-Session-Token", "synthetic-session")
		case strings.HasSuffix(req.URL.Path, "/accountLogin"):
			body = `{"webservices":{"ckdatabasews":{"status":"active","url":"https://p01-ckdatabasews.icloud.com"}}}`
		}
		return &http.Response{StatusCode: 200, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	err := s.Login(context.Background(), "synthetic-account", "synthetic-password", func(context.Context, string, bool) (string, error) { t.Fatal("unexpected challenge"); return "", nil }, false)
	if err != nil || calls != 5 || s.service == "" {
		t.Fatal("synthetic login failed", err, calls)
	}
}

// TestCatalogPagination verifies rank units, cross-page master joins and fresh lookup ownership.
func TestCatalogPagination(t *testing.T) {
	s := New()
	s.service = "https://p01-ckdatabasews.icloud.com/database/1/com.apple.photos.cloud/production/private/"
	pages := 0
	s.client = doer(func(req *http.Request) (*http.Response, error) {
		var payload map[string]jsontext.Value
		if err := json.UnmarshalRead(req.Body, &payload); err != nil {
			t.Fatal(err)
		}
		body := ""
		if strings.HasSuffix(req.URL.Path, "/lookup") {
			body = `{"records":[{"recordName":"master","fields":{"resOriginalRes":{"value":{"downloadURL":"https://cvws.icloud-content.com/synthetic"}}}}]}`
		} else if strings.Contains(string(payload["query"]), "CheckIndexingState") {
			body = `{"records":[{"fields":{"state":{"value":"FINISHED"}}}]}`
		} else {
			pages++
			switch pages {
			case 1:
				body = `{"records":[{"recordName":"asset-a","recordType":"CPLAsset","fields":{"masterRef":{"value":{"recordName":"master"}}}}]}`
			case 2:
				if !strings.Contains(string(payload["query"]), `"value":1`) {
					t.Fatal("rank did not advance by assets")
				}
				body = `{"records":[{"recordName":"master","recordType":"CPLMaster","fields":{"filenameEnc":{"value":"synthetic.jpg","type":"STRING"},"resOriginalRes":{"value":{"size":3}}}},{"recordName":"asset-b","recordType":"CPLAsset","fields":{"masterRef":{"value":{"recordName":"master"}}}}]}`
			default:
				body = `{"records":[]}`
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	assets, err := s.List(context.Background())
	if err != nil || len(assets) != 2 || !assets[0].Supported || !assets[1].Supported {
		t.Fatal(assets, err)
	}
	raw, err := s.URL(context.Background(), archive.Asset{ID: "asset-b"})
	if err != nil || raw == "" {
		t.Fatal("fresh lookup failed", err)
	}
	for _, bad := range []string{"https://icloud-content.com.evil.invalid/a", "http://cvws.icloud-content.com/a", "https://user:pass@cvws.icloud-content.com/a"} {
		if mediaAuthority(bad) {
			t.Fatal("unsafe media authority")
		}
	}
}

// TestVerificationConsent rejects declined delivery without trust or code operations.
func TestVerificationConsent(t *testing.T) {
	s := New()
	verificationCalls := 0
	s.client = doer(func(req *http.Request) (*http.Response, error) {
		body := "{}"
		status := 200
		switch {
		case strings.HasSuffix(req.URL.Path, "/signin/init"):
			body = `{"iteration":100,"salt":"c3ludGhldGlj","protocol":"s2k","b":"Ag==","c":"synthetic"}`
		case strings.HasSuffix(req.URL.Path, "/signin/complete"):
			status = 409
		case strings.Contains(req.URL.Path, "/verify/") || strings.Contains(req.URL.Path, "/trust"):
			verificationCalls++
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	err := s.Login(context.Background(), "synthetic-account", "synthetic-password", func(context.Context, string, bool) (string, error) { return "n", nil }, true)
	if err == nil || verificationCalls != 0 {
		t.Fatal("declined verification performed account action")
	}
}

// TestVerificationDeliveryResponse keeps delivery separate from code authentication.
func TestVerificationDeliveryResponse(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		delivery, verification  int
		wantPrompt, wantSuccess bool
	}{
		{"delivered", 200, 200, true, true},
		{"accepted", 202, 200, true, true},
		{"no-content", 204, 200, true, true},
		{"challenge-outstanding", 409, 200, true, true},
		{"wrong-code", 409, 403, true, false},
		{"throttled", 429, 200, false, false},
		{"denied", 403, 200, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			codePrompts, codePosts, accountLogins, trustCalls := 0, 0, 0, 0
			s.client = doer(func(req *http.Request) (*http.Response, error) {
				body := "{}"
				status := 200
				headers := make(http.Header)
				switch {
				case strings.HasSuffix(req.URL.Path, "/signin/init"):
					body = `{"iteration":100,"salt":"c3ludGhldGlj","protocol":"s2k","b":"Ag==","c":"synthetic"}`
				case strings.HasSuffix(req.URL.Path, "/signin/complete"):
					status = 409
				case strings.HasSuffix(req.URL.Path, "/verify/trusteddevice/securitycode"):
					if req.Method == http.MethodPut {
						status = tc.delivery
					} else {
						codePosts++
						status = tc.verification
						var payload struct {
							SecurityCode struct {
								Code string `json:"code"`
							} `json:"securityCode"`
						}
						if json.UnmarshalRead(req.Body, &payload) != nil || payload.SecurityCode.Code != "123456" {
							t.Fatal("code not submitted for normal verification")
						}
						if status == 200 {
							headers.Set("X-Apple-Session-Token", "synthetic-session")
						}
					}
				case strings.HasSuffix(req.URL.Path, "/2sv/trust"):
					trustCalls++
				case strings.HasSuffix(req.URL.Path, "/accountLogin"):
					accountLogins++
					body = `{"webservices":{"ckdatabasews":{"status":"active","url":"https://p01-ckdatabasews.icloud.com"}}}`
				}
				return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			prompt := func(_ context.Context, label string, secret bool) (string, error) {
				if !secret {
					return "y", nil
				}
				codePrompts++
				return "123456", nil // Synthetic code, never an account capture.
			}
			err := s.Login(context.Background(), "synthetic-account", "synthetic-password", prompt, false)
			if (err == nil) != tc.wantSuccess {
				t.Fatalf("unexpected login outcome: %v", err)
			}
			expected := 0
			if tc.wantPrompt {
				expected = 1
			}
			if codePrompts != expected || codePosts != expected {
				t.Fatalf("code prompts/posts = %d/%d, expected %d", codePrompts, codePosts, expected)
			}
			if trustCalls != 0 {
				t.Fatal("automatic trust operation")
			}
			if !tc.wantSuccess && accountLogins != 0 {
				t.Fatal("account login before code accepted")
			}
		})
	}
}

// TestTrustHandoff requires explicit consent and passes only the returned ephemeral trust token.
func TestTrustHandoff(t *testing.T) {
	for _, tc := range []struct {
		name, consent              string
		trustStatus, accountStatus int
		wantSuccess                bool
		wantStage                  string
	}{
		{"trust-ok", "y", 200, 200, true, ""},
		{"trust-no-content", "y", 204, 200, true, ""},
		{"trust-denied", "y", 403, 200, false, "session trust failed (HTTP 403)"},
		{"account-denied", "y", 204, 401, false, "account login failed after verification (HTTP 401)"},
		{"declined-trust", "n", 200, 200, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			trustCalls, accountCalls := 0, 0
			s.client = doer(func(req *http.Request) (*http.Response, error) {
				status := 200
				body := "{}"
				headers := make(http.Header)
				switch {
				case strings.HasSuffix(req.URL.Path, "/signin/init"):
					body = `{"iteration":100,"salt":"c3ludGhldGlj","protocol":"s2k","b":"Ag==","c":"synthetic"}`
				case strings.HasSuffix(req.URL.Path, "/signin/complete"):
					status = 409
				case strings.HasSuffix(req.URL.Path, "/verify/trusteddevice/securitycode"):
					if req.Method == http.MethodPost {
						headers.Set("X-Apple-Session-Token", "synthetic-session")
					}
				case strings.HasSuffix(req.URL.Path, "/2sv/trust"):
					trustCalls++
					status = tc.trustStatus
					if status == 200 || status == 204 {
						headers.Set("X-Apple-TwoSV-Trust-Token", "synthetic-trust-token")
					}
				case strings.HasSuffix(req.URL.Path, "/accountLogin"):
					accountCalls++
					status = tc.accountStatus
					var payload map[string]jsontext.Value
					if err := json.UnmarshalRead(req.Body, &payload); err != nil {
						t.Fatal(err)
					}
					if tc.consent == "y" {
						if string(payload["trustToken"]) != `"synthetic-trust-token"` {
							t.Fatal("approved trust token not handed off")
						}
					} else if _, present := payload["trustToken"]; present {
						t.Fatal("trust state used without consent")
					}
					body = `{"webservices":{"ckdatabasews":{"status":"active","url":"https://p01-ckdatabasews.icloud.com"}}}`
				}
				return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			prompt := func(_ context.Context, label string, secret bool) (string, error) {
				if secret {
					return "123456", nil
				}
				if strings.Contains(label, "session trust") {
					return tc.consent, nil
				}
				return "y", nil
			}
			err := s.Login(context.Background(), "synthetic-account", "synthetic-password", prompt, true)
			if (err == nil) != tc.wantSuccess {
				t.Fatalf("unexpected result: %v", err)
			}
			if tc.wantStage != "" && (err == nil || !strings.Contains(err.Error(), tc.wantStage)) {
				t.Fatalf("missing safe stage: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "synthetic-trust-token") {
				t.Fatal("token exposed")
			}
			if tc.consent == "n" && trustCalls != 0 {
				t.Fatal("trust without consent")
			}
			if tc.trustStatus == 403 && accountCalls != 0 {
				t.Fatal("account login after failed trust")
			}
		})
	}
}

// TestPhotosEndpoint covers optional status, default TLS ports and privacy-safe rejection reasons.
func TestPhotosEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, raw, status string
		present, ok       bool
		reason            string
	}{
		{"active", "https://p01-ckdatabasews.icloud.com", "active", true, true, ""},
		{"missing-status", "https://p01-ckdatabasews.icloud.com", "", true, true, ""},
		{"explicit-tls", "https://p01-ckdatabasews.icloud.com:443/", "active", true, true, ""},
		{"missing-service", "", "", false, false, "did not return ckdatabasews"},
		{"missing-url", "", "active", true, false, "no service URL"},
		{"inactive", "https://p01-ckdatabasews.icloud.com", "inactive", true, false, "service status"},
		{"other-apple-host", "https://p01-other.icloud.com", "active", true, false, "supported Photos authority pattern"},
		{"evil-suffix", "https://p01-ckdatabasews.icloud.com.evil.invalid", "active", true, false, "outside"},
		{"http", "http://p01-ckdatabasews.icloud.com", "active", true, false, "HTTPS"},
		{"wrong-port", "https://p01-ckdatabasews.icloud.com:8443", "active", true, false, "unapproved port"},
		{"signed-query", "https://p01-ckdatabasews.icloud.com/?token=SECRET", "active", true, false, "query or fragment"},
		{"userinfo", "https://SECRET@p01-ckdatabasews.icloud.com", "active", true, false, "user information"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, err := photosEndpoint(tc.raw, tc.status, tc.present)
			if (err == nil) != tc.ok {
				t.Fatalf("unexpected result: %v", err)
			}
			if tc.ok {
				src := New()
				src.service = service
				if !src.allowedAPI(service + "records/query") {
					t.Fatal("normalized service not admitted")
				}
			} else {
				if !strings.Contains(err.Error(), tc.reason) {
					t.Fatalf("reason missing: %v", err)
				}
				if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), tc.raw) && tc.raw != "" {
					t.Fatal("service URL exposed")
				}
			}
		})
	}
}
