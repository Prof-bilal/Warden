package mcpbridge

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"
)

type Auth struct {
	Token    string
	Issuer   string
	Audience string
	Scope    string
	keys     map[string]*rsa.PublicKey
}

func LoadAuth(token, issuer, audience, jwksFile string) (*Auth, error) {
	a := &Auth{Token: token, Issuer: issuer, Audience: audience, Scope: "mcp"}
	if token != "" {
		if len(token) < 32 || issuer != "" || jwksFile != "" {
			return nil, fmt.Errorf("bearer token must have 32+ characters and cannot mix with OAuth")
		}
		return a, nil
	}
	u, e := url.Parse(issuer)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || audience == "" || jwksFile == "" {
		return nil, fmt.Errorf("OAuth needs trusted HTTPS issuer, audience and local JWKS file")
	}
	b, e := os.ReadFile(jwksFile)
	if e != nil {
		return nil, e
	}
	if len(b) > 1024*1024 || StrictJSON(b) != nil {
		return nil, fmt.Errorf("invalid JWKS")
	}
	var set struct {
		Keys []struct{ Kty, Use, Alg, Kid, N, E string } `json:"keys"`
	}
	if json.Unmarshal(b, &set) != nil {
		return nil, fmt.Errorf("invalid JWKS")
	}
	a.keys = map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || k.Kid == "" || (k.Alg != "" && k.Alg != "RS256") || (k.Use != "" && k.Use != "sig") {
			continue
		}
		n, e := base64.RawURLEncoding.DecodeString(k.N)
		if e != nil {
			return nil, e
		}
		exp, e := base64.RawURLEncoding.DecodeString(k.E)
		if e != nil || len(exp) > 4 {
			return nil, fmt.Errorf("invalid RSA exponent")
		}
		v := 0
		for _, x := range exp {
			v = v*256 + int(x)
		}
		if v < 3 || v%2 == 0 || len(n) < 256 || a.keys[k.Kid] != nil {
			return nil, fmt.Errorf("weak or duplicate RSA key")
		}
		a.keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: v}
	}
	if len(a.keys) == 0 {
		return nil, fmt.Errorf("no trusted RS256 signing keys")
	}
	return a, nil
}
func (a *Auth) Identity(header string) (string, error) {
	if !strings.HasPrefix(header, "Bearer ") || len(header) > 16384 {
		return "", fmt.Errorf("unauthorized")
	}
	token := strings.TrimPrefix(header, "Bearer ")
	if a.Token != "" {
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.Token)) != 1 {
			return "", fmt.Errorf("unauthorized")
		}
		return Hash([]byte(token)), nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid access token")
	}
	decode := func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
	h, e := decode(parts[0])
	if e != nil || StrictJSON(h) != nil {
		return "", fmt.Errorf("invalid JWT header")
	}
	var head struct {
		Alg, Kid string
		Crit     []string
	}
	if json.Unmarshal(h, &head) != nil || head.Alg != "RS256" || len(head.Crit) > 0 {
		return "", fmt.Errorf("unsupported JWT algorithm")
	}
	key := a.keys[head.Kid]
	if key == nil {
		return "", fmt.Errorf("untrusted issuer key")
	}
	sig, e := decode(parts[2])
	if e != nil {
		return "", e
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig) != nil {
		return "", fmt.Errorf("invalid access-token signature")
	}
	p, e := decode(parts[1])
	if e != nil || StrictJSON(p) != nil {
		return "", fmt.Errorf("invalid JWT claims")
	}
	var c struct {
		Issuer    string          `json:"iss"`
		Subject   string          `json:"sub"`
		Audience  json.RawMessage `json:"aud"`
		Expires   int64           `json:"exp"`
		NotBefore int64           `json:"nbf"`
		Scope     string          `json:"scope"`
	}
	if json.Unmarshal(p, &c) != nil || c.Issuer != a.Issuer || c.Subject == "" || c.Expires <= time.Now().Unix() || c.NotBefore > time.Now().Unix() {
		return "", fmt.Errorf("invalid token claims")
	}
	var audiences []string
	var one string
	if json.Unmarshal(c.Audience, &one) == nil {
		audiences = []string{one}
	} else {
		_ = json.Unmarshal(c.Audience, &audiences)
	}
	if !contains(audiences, a.Audience) || !contains(strings.Fields(c.Scope), a.Scope) {
		return "", fmt.Errorf("wrong audience or missing mcp scope")
	}
	return Hash([]byte(c.Issuer + "\x00" + c.Subject)), nil
}
