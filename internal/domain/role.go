package domain

// Role is a console RBAC role stored on users.
type Role string

const (
	RoleAdministrator Role = "Administrator"
	RoleOperator      Role = "Operator"
	RoleAuditor       Role = "Auditor"
	RoleGuest         Role = "Guest"
)

// Valid reports whether r is one of the four supported roles.
func (r Role) Valid() bool {
	switch r {
	case RoleAdministrator, RoleOperator, RoleAuditor, RoleGuest:
		return true
	default:
		return false
	}
}
