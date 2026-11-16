// Package authz holds who is calling (Actor) and what each gRPC method allows
// (Policy). The policy is a declarative map per service, like manabie's
// rbacDecider, but deny-by-default: a method without a rule is refused, and
// Policy.Validate fails startup when a new RPC is added without one.
package authz

import (
	"fmt"
	"slices"
	"sort"

	"google.golang.org/grpc"
)

// Roles carried in the access token. Values match users.role in auth_db.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Actor is the authenticated caller, passed explicitly into application code.
type Actor struct {
	UserID string
	Role   string
}

func (a Actor) IsAdmin() bool { return a.Role == RoleAdmin }

// Rule says who may call one method.
type Rule struct {
	public bool
	roles  []string // empty + !public = any authenticated user
}

// Public methods need no token (login, register, ...).
var Public = Rule{public: true}

// Authenticated allows any caller with a valid token.
var Authenticated = Rule{}

// Roles allows only callers whose token role is in roles.
func Roles(roles ...string) Rule { return Rule{roles: roles} }

func (r Rule) IsPublic() bool { return r.public }

func (r Rule) Allows(role string) bool {
	return len(r.roles) == 0 || slices.Contains(r.roles, role)
}

// Policy maps full gRPC method names ("/pkg.Service/Method") to rules.
type Policy map[string]Rule

// Validate returns an error listing every method of the services that has no
// rule. Call it at startup so a forgotten rule fails fast instead of being
// silently denied (or worse, silently allowed) in production.
func (p Policy) Validate(services ...grpc.ServiceDesc) error {
	var missing []string
	for _, sd := range services {
		for _, m := range sd.Methods {
			full := "/" + sd.ServiceName + "/" + m.MethodName
			if _, ok := p[full]; !ok {
				missing = append(missing, full)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("authz policy has no rule for: %v", missing)
	}
	return nil
}
