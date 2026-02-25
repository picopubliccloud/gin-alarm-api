// /internal/ticketing/auth/verifier.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type Verifier struct {
	Issuer     string
	JWKSURL    string
	FetchEvery time.Duration

	mu        sync.RWMutex
	keySet    jwk.Set
	lastFetch time.Time
}

// NewVerifier fetches JWKS once at startup and refreshes periodically.
// This assumes your Keycloak TLS cert is now trusted by the OS CA store and has proper SAN.
func NewVerifier(jwksURL, issuer string) (*Verifier, error) {
	v := &Verifier{
		Issuer:     issuer,
		JWKSURL:    jwksURL,
		FetchEvery: 30 * time.Minute,
	}

	if err := v.refresh(context.Background()); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Verifier) refresh(ctx context.Context) error {
	ks, err := jwk.Fetch(ctx, v.JWKSURL)
	if err != nil {
		return fmt.Errorf("jwks fetch failed: %w", err)
	}

	v.mu.Lock()
	v.keySet = ks
	v.lastFetch = time.Now()
	v.mu.Unlock()

	return nil
}

func (v *Verifier) maybeRefresh(ctx context.Context) {
	v.mu.RLock()
	ksNil := v.keySet == nil
	age := time.Since(v.lastFetch)
	v.mu.RUnlock()

	if ksNil || age > v.FetchEvery {
		_ = v.refresh(ctx) // keep old keys if refresh fails
	}
}

// Keyfunc is compatible with github.com/golang-jwt/jwt/v5
func (v *Verifier) Keyfunc(token *jwt.Token) (any, error) {
	v.maybeRefresh(context.Background())

	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, errors.New("token header missing kid")
	}

	v.mu.RLock()
	ks := v.keySet
	v.mu.RUnlock()

	if ks == nil {
		return nil, errors.New("jwks not loaded")
	}

	key, ok := ks.LookupKeyID(kid)
	if !ok {
		// Key rotation: refresh and retry once
		_ = v.refresh(context.Background())

		v.mu.RLock()
		ks = v.keySet
		v.mu.RUnlock()

		if ks == nil {
			return nil, errors.New("jwks not loaded after refresh")
		}
		key, ok = ks.LookupKeyID(kid)
		if !ok {
			return nil, fmt.Errorf("kid not found in jwks: %s", kid)
		}
	}

	var pub any
	if err := key.Raw(&pub); err != nil {
		return nil, fmt.Errorf("failed to get raw key: %w", err)
	}
	return pub, nil
}
