// internal/ticketing/auth/roles.go
package auth

import (
	"strings"
	"fmt"
)

func AllRoles(c *KCClaims) []string {
	seen := map[string]struct{}{}
	var out []string

	add := func(r string) {
		r = strings.TrimSpace(r)
		if r == "" {
			return
		}
		key := strings.ToLower(r)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}

	for _, r := range c.RealmAccess.Roles {
		add(r)
	}
	for _, cl := range c.ResourceAccess {
		for _, r := range cl.Roles {
			add(r)
		}
	}
	return out
}

// func InferActorType(roles []string) string {
// 	lower := make([]string, 0, len(roles))
// 	for _, r := range roles {
// 		lower = append(lower, strings.ToLower(r))
// 	}
// 	for _, r := range lower {
// 		if r == "ops" || strings.HasPrefix(r, "ops_") || r == "noc" {
// 			return "OPS"
// 		}
// 	}
// 	return "CUSTOMER"
// }

func InferActorType(roles []string) string {
	// Normalize roles: lowercase + keep only [a-z0-9_] so "P&S" becomes "ps"
	normalize := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return ""
		}
		var b strings.Builder
		b.Grow(len(s))
		for _, ch := range s {
			switch {
			case ch >= 'a' && ch <= 'z':
				b.WriteRune(ch)
			case ch >= '0' && ch <= '9':
				b.WriteRune(ch)
			case ch == '_' || ch == '-':
				b.WriteByte('_') // treat - as _
			// drop everything else (spaces, &, /, etc)
			}
		}
		return b.String()
	}

	// Actor-type rules:
	// - OPS: ops, noc, ops_*
	// - CAREOPS: careops (and care_ops)
	// - PS: p&s -> ps (and ps)
	// - OPS_NETWORK: ops_network
	// - OPS_SECURITY: ops_security
	for _, r := range roles {
		n := normalize(r)
		if n == "" {
			continue
		}
		fmt.Println(n)

		switch {
		case n == "careops" || n == "care_ops":
			return "CAREOPS"

		case n == "ps" || n == "p_s":
			return "P&S"

		case n == "ops_network":
			return "OPS_NETWORK"

		case n == "ops_security":
			return "OPS_SECURITY"

		case n == "ops" || n == "noc" || strings.HasPrefix(n, "ops_"):
			// keep this after the specific ops_* matches above
			return "OPS"
		}
	}

	return ""
}
