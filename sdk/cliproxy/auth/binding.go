package auth

import "strings"

// ResolveBindingIndexes converts persisted binding references into current auth_index values.
// auth_identity wins over auth_index because it survives auth file rewrites.
func ResolveBindingIndexes(auths []*Auth, indexBindings, identityBindings map[string]string, defaultRef string) (map[string]string, string) {
	identityToIndex := StableIdentityIndexMap(auths)
	resolved := make(map[string]string, len(indexBindings)+len(identityBindings))
	for clientKey, authIndex := range indexBindings {
		clientKey = strings.TrimSpace(clientKey)
		authIndex = strings.TrimSpace(authIndex)
		if clientKey != "" && authIndex != "" {
			resolved[clientKey] = authIndex
		}
	}
	for clientKey, identity := range identityBindings {
		clientKey = strings.TrimSpace(clientKey)
		identity = strings.TrimSpace(identity)
		if clientKey == "" || identity == "" {
			continue
		}
		if authIndex := identityToIndex[identity]; authIndex != "" {
			resolved[clientKey] = authIndex
		}
	}
	if len(resolved) == 0 {
		resolved = nil
	}
	return resolved, resolveDefaultBindingRef(strings.TrimSpace(defaultRef), identityToIndex)
}

// StableIdentityIndexMap maps each durable identity to the current runtime auth_index.
// Duplicate identities are ignored rather than guessed; account binding must be deterministic.
func StableIdentityIndexMap(auths []*Auth) map[string]string {
	if len(auths) == 0 {
		return nil
	}
	out := make(map[string]string, len(auths))
	duplicates := make(map[string]struct{})
	for _, auth := range auths {
		if auth == nil {
			continue
		}
		identity := strings.TrimSpace(auth.StableIdentity())
		index := strings.TrimSpace(auth.EnsureIndex())
		if identity == "" || index == "" {
			continue
		}
		if existing := out[identity]; existing != "" && existing != index {
			delete(out, identity)
			duplicates[identity] = struct{}{}
			continue
		}
		if _, duplicate := duplicates[identity]; !duplicate {
			out[identity] = index
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func resolveDefaultBindingRef(ref string, identityToIndex map[string]string) string {
	if ref == "" {
		return ""
	}
	if authIndex := identityToIndex[ref]; authIndex != "" {
		return authIndex
	}
	if strings.Contains(ref, ":") {
		return ""
	}
	return ref
}
