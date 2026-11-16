package authz

import (
	"strings"
	"testing"

	"google.golang.org/grpc"
)

func TestPolicyValidate(t *testing.T) {
	sd := grpc.ServiceDesc{ServiceName: "pkg.Svc", Methods: []grpc.MethodDesc{{MethodName: "A"}, {MethodName: "B"}}}

	if err := (Policy{"/pkg.Svc/A": Public, "/pkg.Svc/B": Authenticated}).Validate(sd); err != nil {
		t.Fatalf("complete policy: %v", err)
	}
	err := Policy{"/pkg.Svc/A": Public}.Validate(sd)
	if err == nil || !strings.Contains(err.Error(), "/pkg.Svc/B") {
		t.Fatalf("expected missing /pkg.Svc/B, got %v", err)
	}
}

func TestRuleAllows(t *testing.T) {
	if !Authenticated.Allows("anything") {
		t.Fatal("Authenticated must allow any role")
	}
	r := Roles(RoleAdmin)
	if !r.Allows(RoleAdmin) || r.Allows(RoleUser) {
		t.Fatal("Roles(admin) misbehaves")
	}
}
