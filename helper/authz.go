package helper

import "sort"

type PermissionSet struct {
	byRole map[string]map[string]struct{}
}

func NewPermissionSet(mapping map[string][]string) *PermissionSet {
	ps := &PermissionSet{byRole: make(map[string]map[string]struct{})}

	for role, permissions := range mapping {
		ps.byRole[role] = make(map[string]struct{})
		for _, permission := range permissions {
			ps.byRole[role][permission] = struct{}{}
		}
	}

	return ps
}

func (ps *PermissionSet) Can(role, permission string) bool {
	if ps == nil || ps.byRole == nil {
		return false
	}

	permissions, exists := ps.byRole[role]
	if !exists {
		return false
	}

	_, allowed := permissions[permission]
	return allowed
}

func (ps *PermissionSet) PermissionsOf(role string) []string {
	if ps == nil {
		return []string{}
	}

	permissions := ps.byRole[role]
	result := make([]string, 0, len(permissions))
	for permission := range permissions {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func (ps *PermissionSet) KnownRoles() []string {
	if ps == nil {
		return []string{}
	}

	result := make([]string, 0, len(ps.byRole))
	for role := range ps.byRole {
		result = append(result, role)
	}
	sort.Strings(result)
	return result
}

func (ps *PermissionSet) IsKnownRole(role string) bool {
	if ps == nil {
		return false
	}

	_, exists := ps.byRole[role]
	return exists
}
