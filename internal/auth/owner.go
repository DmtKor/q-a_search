package auth

import "strings"

// IsOwner returns true if the principal is the owner of the resource (e.g. case),
// by comparing principal.TokenID with resource_created_by (both stored as strings; normalize for comparison).
func IsOwner(principal *Principal, resourceCreatedBy string) bool {
	if principal == nil || resourceCreatedBy == "" {
		return false
	}
	return strings.TrimSpace(strings.ToLower(principal.TokenID)) ==
		strings.TrimSpace(strings.ToLower(resourceCreatedBy))
}
