// Package rbac package rbac
package rbac

import (
	"context"
	"embed"

	"github.com/open-policy-agent/opa/v1/rego"
)

//go:embed role.rego
var role embed.FS

type Action struct {
	Method string
	Path   string
	Role   string
}

func Check(action Action) bool {
	roleContent, err := role.ReadFile("role.rego")
	if err != nil {
		return false
	}

	ctx := context.Background()
	query, err := rego.New(
		rego.Query("x = data.rbac.allow"),
		rego.Module("rbac.rego", string(roleContent)),
	).PrepareForEval(ctx)
	if err != nil {
		return false
	}

	input := map[string]any{
		"method": action.Method,
		"path":   action.Path,
		"role":   action.Role,
	}

	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false
	} else if len(results) == 0 {
		return false
	}
	result, ok := results[0].Bindings["x"].(bool)
	if !ok {
		return false
	}
	return result
}
