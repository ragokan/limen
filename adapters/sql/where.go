package sql

import (
	"fmt"
	"strings"

	"github.com/ragokan/limen"
)

var sqlComparisonOps = map[limen.Operator]string{
	"":          "=",
	limen.OpEq:  "=",
	limen.OpNe:  "!=",
	limen.OpLt:  "<",
	limen.OpLte: "<=",
	limen.OpGt:  ">",
	limen.OpGte: ">=",
}

// buildWhere returns a WHERE clause (without "WHERE") and args. Uses ? placeholders; caller must rebind before execute.
// Conditions are grouped by OR runs; groups are then AND'd. E.g. [A, B OR C, D] -> (A) AND (B OR C) AND (D).
func (a *Adapter) buildWhere(conditions []limen.Where) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}

	if len(conditions) == 1 {
		return a.buildOneCondition(conditions[0])
	}

	groups := limen.GroupConditionsByConnector(conditions)
	parts := make([]string, 0, len(groups))
	var args []any
	for _, group := range groups {
		clause, groupArgs, err := a.buildGroupClause(group)
		if err != nil {
			return "", nil, err
		}
		if clause == "" {
			continue
		}
		if len(group) > 1 {
			clause = "(" + clause + ")"
		}
		parts = append(parts, clause)
		args = append(args, groupArgs...)
	}
	return strings.Join(parts, " AND "), args, nil
}

func (a *Adapter) buildGroupClause(group []limen.Where) (string, []any, error) {
	clauses := make([]string, 0, len(group))
	var args []any
	for _, c := range group {
		clause, clauseArgs, err := a.buildOneCondition(c)
		if err != nil {
			return "", nil, err
		}
		if clause == "" {
			continue
		}
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}
	return strings.Join(clauses, " OR "), args, nil
}

func (a *Adapter) buildOneCondition(c limen.Where) (string, []any, error) {
	col := a.quoteIdent(c.Column)
	const ph = "?"

	switch c.Operator {
	case limen.OpEq, "", limen.OpNe, limen.OpLt, limen.OpLte, limen.OpGt, limen.OpGte:
		return col + " " + sqlComparisonOps[c.Operator] + " " + ph, []any{c.Value}, nil
	case limen.OpIn:
		return a.collectionCondition(col, c.Value, "IN", "1 = 0")
	case limen.OpNotIn:
		return a.collectionCondition(col, c.Value, "NOT IN", "1 = 1")
	case limen.OpContains:
		return likeCondition(col, c.Value, "contains", "%", "%")
	case limen.OpStartsWith:
		return likeCondition(col, c.Value, "starts_with", "", "%")
	case limen.OpEndsWith:
		return likeCondition(col, c.Value, "ends_with", "%", "")
	case limen.OpIsNull:
		return col + " IS NULL", nil, nil
	case limen.OpIsNotNull:
		return col + " IS NOT NULL", nil, nil
	default:
		return "", nil, fmt.Errorf("%w: unsupported operator %q", limen.ErrInvalidCondition, c.Operator)
	}
}

func (a *Adapter) collectionCondition(col string, value any, operator, emptyClause string) (string, []any, error) {
	vals, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: %s requires []any", limen.ErrInvalidCondition, operator)
	}
	if len(vals) == 0 {
		return emptyClause, nil, nil
	}
	placeholders := strings.Repeat("?, ", len(vals)-1) + "?"
	return col + " " + operator + " (" + placeholders + ")", vals, nil
}

func likeCondition(col string, value any, operator, prefix, suffix string) (string, []any, error) {
	s, ok := value.(string)
	if !ok {
		return "", nil, fmt.Errorf("%w: %s requires string", limen.ErrInvalidCondition, operator)
	}
	return col + " LIKE ? ESCAPE '\\'", []any{prefix + escapeLike(s) + suffix}, nil
}

func validateOrderByDirection(direction limen.OrderByDirection) error {
	if direction == limen.OrderByAsc || direction == limen.OrderByDesc {
		return nil
	}
	return fmt.Errorf("%w: unsupported order direction %q", limen.ErrInvalidCondition, direction)
}

// escapeLike escapes %, _, and \ for use in LIKE patterns with ESCAPE '\\'.
func escapeLike(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
