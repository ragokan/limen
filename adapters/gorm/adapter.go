package gorm

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ragokan/limen"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	comparisonOps     = map[limen.Operator]string{
		"":          "=",
		limen.OpEq:  "=",
		limen.OpNe:  "!=",
		limen.OpLt:  "<",
		limen.OpLte: "<=",
		limen.OpGt:  ">",
		limen.OpGte: ">=",
	}
)

// Adapter implements limen.DatabaseAdapter using GORM
type Adapter struct {
	db *gorm.DB // Regular DB connection
	tx *gorm.DB // Transaction DB (nil when not in transaction)
}

// New creates a new GORM adapter
func New(db *gorm.DB) *Adapter {
	return &Adapter{db: db}
}

// getDB returns the transaction DB if in a transaction, otherwise returns the regular DB
func (a *Adapter) getDB() *gorm.DB {
	if a.tx != nil {
		return a.tx
	}
	return a.db
}

// BeginTx starts a new transaction
func (a *Adapter) BeginTx(ctx context.Context) (limen.DatabaseTx, error) {
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &Adapter{
		db: a.db,
		tx: tx,
	}, nil
}

// Commit commits the transaction if this adapter is in transaction mode
func (a *Adapter) Commit() error {
	if a.tx == nil {
		return fmt.Errorf("not in a transaction")
	}
	err := a.tx.Commit().Error
	a.tx = nil
	return err
}

// Rollback rolls back the transaction if this adapter is in transaction mode
func (a *Adapter) Rollback() error {
	if a.tx == nil {
		return fmt.Errorf("not in a transaction")
	}
	err := a.tx.Rollback().Error
	a.tx = nil // Clear transaction state
	return err
}

func (a *Adapter) Create(ctx context.Context, tableName limen.SchemaTableName, data map[string]any) error {
	db := a.getDB()
	return db.WithContext(ctx).Table(string(tableName)).Create(data).Error
}

func (a *Adapter) FindOne(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, orderBy []limen.OrderBy) (map[string]any, error) {
	var result map[string]any
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))

	query, err := a.applyConditions(query, conditions)
	if err != nil {
		return nil, err
	}

	for _, orderBy := range orderBy {
		query, err = applyOrderBy(query, orderBy)
		if err != nil {
			return nil, err
		}
	}

	err = query.Take(&result).Error
	return result, a.formatError(err)
}

func (a *Adapter) FindMany(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, options *limen.QueryOptions) ([]map[string]any, error) {
	var results []map[string]any
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))

	query, err := a.applyConditions(query, conditions)
	if err != nil {
		return nil, err
	}

	if options != nil {
		if options.Limit > 0 {
			query = query.Limit(options.Limit)
		}
		if options.Offset > 0 {
			query = query.Offset(options.Offset)
		}
		for _, orderBy := range options.OrderBy {
			query, err = applyOrderBy(query, orderBy)
			if err != nil {
				return nil, err
			}
		}
	}

	err = query.Find(&results).Error
	return results, a.formatError(err)
}

func (a *Adapter) Update(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	if len(conditions) == 0 {
		return fmt.Errorf("%w: conditions required to prevent accidental table-wide update", limen.ErrMissingConditions)
	}

	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query, err := a.applyConditions(query, conditions)
	if err != nil {
		return err
	}
	return query.Updates(updates).Error
}

func (a *Adapter) Delete(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) error {
	if len(conditions) == 0 {
		return fmt.Errorf("%w: conditions required to prevent accidental table-wide delete", limen.ErrMissingConditions)
	}

	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query, err := a.applyConditions(query, conditions)
	if err != nil {
		return err
	}
	return query.Delete(nil).Error
}

func (a *Adapter) Exists(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) (bool, error) {
	var result map[string]any
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query, err := a.applyConditions(query, conditions)
	if err != nil {
		return false, err
	}
	err = query.Select("1").Limit(1).Take(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (a *Adapter) Count(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) (int64, error) {
	var count int64
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query, err := a.applyConditions(query, conditions)
	if err != nil {
		return 0, err
	}
	err = query.Count(&count).Error
	return count, err
}

func (a *Adapter) applyConditions(query *gorm.DB, conditions []limen.Where) (*gorm.DB, error) {
	if len(conditions) == 0 {
		return query, nil
	}

	if len(conditions) == 1 {
		whereClause, args, err := a.buildWhereClause(conditions[0])
		if err != nil {
			return nil, err
		}
		if whereClause == "" {
			return query, nil
		}
		return query.Where(whereClause, args...), nil
	}

	groups := limen.GroupConditionsByConnector(conditions)
	for _, group := range groups {
		var err error
		query, err = a.applyGroup(query, group)
		if err != nil {
			return nil, err
		}
	}
	return query, nil
}

// applyGroup applies one group (single condition or OR of several) as one Where.
func (a *Adapter) applyGroup(query *gorm.DB, group []limen.Where) (*gorm.DB, error) {
	if len(group) == 0 {
		return query, nil
	}
	groupClause, args, err := a.buildGroupClause(group)
	if err != nil {
		return nil, err
	}
	if groupClause == "" {
		return query, nil
	}
	return query.Where(groupClause, args...), nil
}

func (a *Adapter) buildGroupClause(group []limen.Where) (string, []any, error) {
	clauses := make([]string, 0, len(group))
	var args []any
	for _, condition := range group {
		whereClause, clauseArgs, err := a.buildWhereClause(condition)
		if err != nil {
			return "", nil, err
		}
		if whereClause == "" {
			continue
		}
		clauses = append(clauses, whereClause)
		args = append(args, clauseArgs...)
	}
	if len(clauses) == 0 {
		return "", nil, nil
	}
	if len(clauses) == 1 {
		return clauses[0], args, nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args, nil
}

func (a *Adapter) buildWhereClause(condition limen.Where) (string, []any, error) {
	column, err := safeColumn(condition.Column)
	if err != nil {
		return "", nil, err
	}
	switch condition.Operator {
	case limen.OpEq, "", limen.OpNe, limen.OpLt, limen.OpLte, limen.OpGt, limen.OpGte:
		return column + " " + comparisonOps[condition.Operator] + " ?", []any{condition.Value}, nil
	case limen.OpIn:
		return collectionWhereClause(column, condition.Value, "IN", "1 = 0")
	case limen.OpNotIn:
		return collectionWhereClause(column, condition.Value, "NOT IN", "1 = 1")
	case limen.OpContains:
		return likeWhereClause(column, condition.Value, "contains", "%", "%")
	case limen.OpStartsWith:
		return likeWhereClause(column, condition.Value, "starts_with", "", "%")
	case limen.OpEndsWith:
		return likeWhereClause(column, condition.Value, "ends_with", "%", "")
	case limen.OpIsNull:
		return column + " IS NULL", nil, nil
	case limen.OpIsNotNull:
		return column + " IS NOT NULL", nil, nil
	default:
		return "", nil, fmt.Errorf("%w: unsupported operator %q", limen.ErrInvalidCondition, condition.Operator)
	}
}

func collectionWhereClause(column string, value any, operator, emptyClause string) (string, []any, error) {
	vals, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: %s requires []any", limen.ErrInvalidCondition, operator)
	}
	if len(vals) == 0 {
		return emptyClause, nil, nil
	}
	return column + " " + operator + " ?", []any{vals}, nil
}

func likeWhereClause(column string, value any, operator, prefix, suffix string) (string, []any, error) {
	s, ok := value.(string)
	if !ok {
		return "", nil, fmt.Errorf("%w: %s requires string", limen.ErrInvalidCondition, operator)
	}
	return column + " LIKE ?", []any{prefix + s + suffix}, nil
}

func applyOrderBy(query *gorm.DB, orderBy limen.OrderBy) (*gorm.DB, error) {
	if orderBy.Direction != limen.OrderByAsc && orderBy.Direction != limen.OrderByDesc {
		return nil, fmt.Errorf("%w: unsupported order direction %q", limen.ErrInvalidCondition, orderBy.Direction)
	}
	if _, err := safeColumn(orderBy.Column); err != nil {
		return nil, err
	}
	return query.Order(clause.OrderByColumn{
		Column: clause.Column{Name: orderBy.Column},
		Desc:   orderBy.Direction == limen.OrderByDesc,
	}), nil
}

func safeColumn(column string) (string, error) {
	if !identifierPattern.MatchString(column) {
		return "", fmt.Errorf("%w: unsafe column %q", limen.ErrInvalidCondition, column)
	}
	return column, nil
}

func (a *Adapter) formatError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return limen.ErrRecordNotFound
	}
	return err
}
