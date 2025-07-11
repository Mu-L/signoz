package sqlschema

import (
	"strings"

	"github.com/SigNoz/signoz/pkg/valuer"
)

var (
	ConstraintTypePrimaryKey = ConstraintType{s: valuer.NewString("pk")}
	ConstraintTypeForeignKey = ConstraintType{s: valuer.NewString("fk")}
	ConstraintTypeCheck      = ConstraintType{s: valuer.NewString("ck")}
	ConstraintTypeUnique     = ConstraintType{s: valuer.NewString("uq")}
)

type ConstraintType struct{ s valuer.String }

func (c ConstraintType) String() string {
	return c.s.String()
}

var (
	_ Constraint = (*PrimaryKeyConstraint)(nil)
	_ Constraint = (*ForeignKeyConstraint)(nil)
	_ Constraint = (*UniqueConstraint)(nil)
)

type Constraint interface {
	// The name of the constraint. This will be autogenerated and should not be set by the user.
	//   - Primary keys are named as `pk_<table_name>`.
	//   - Foreign key constraints are named as `fk_<table_name>_<column_name>`.
	//   - Check constraints are named as `ck_<table_name>_<name>`. The name is the name of the check constraint.
	Name(tableName TableName) string

	// Add name to the constraint. This is typically used to override the autogenerated name because the database might have a different name.
	Named(name string) Constraint

	// The type of the constraint.
	Type() ConstraintType

	// The columns that the constraint is applied to.
	Columns() []ColumnName

	// Equals returns true if the constraint is equal to the other constraint.
	Equals(other Constraint) bool

	// The SQL representation of the constraint.
	ToDefinitionSQL(fmter SQLFormatter, tableName TableName) []byte

	// The SQL representation of the constraint.
	ToDropSQL(fmter SQLFormatter, tableName TableName) []byte
}

type PrimaryKeyConstraint struct {
	ColumnNames []ColumnName
	name        string
}

func (constraint *PrimaryKeyConstraint) Name(tableName TableName) string {
	if constraint.name != "" {
		return constraint.name
	}

	var b strings.Builder
	b.WriteString(ConstraintTypePrimaryKey.String())
	b.WriteString("_")
	b.WriteString(string(tableName))
	return b.String()
}

func (constraint *PrimaryKeyConstraint) Named(name string) Constraint {
	copyOfColumnNames := make([]ColumnName, len(constraint.ColumnNames))
	copy(copyOfColumnNames, constraint.ColumnNames)

	return &PrimaryKeyConstraint{
		ColumnNames: copyOfColumnNames,
		name:        name,
	}
}

func (constraint *PrimaryKeyConstraint) Type() ConstraintType {
	return ConstraintTypePrimaryKey
}

func (constraint *PrimaryKeyConstraint) Columns() []ColumnName {
	return constraint.ColumnNames
}

func (constraint *PrimaryKeyConstraint) Equals(other Constraint) bool {
	if other.Type() != ConstraintTypePrimaryKey {
		return false
	}

	if len(constraint.ColumnNames) != len(other.Columns()) {
		return false
	}

	foundColumns := make(map[ColumnName]bool)
	for _, column := range constraint.ColumnNames {
		foundColumns[column] = true
	}

	for _, column := range other.Columns() {
		if !foundColumns[column] {
			return false
		}
	}

	return true
}

func (constraint *PrimaryKeyConstraint) ToDefinitionSQL(fmter SQLFormatter, tableName TableName) []byte {
	sql := []byte{}

	sql = append(sql, "CONSTRAINT "...)
	sql = fmter.AppendIdent(sql, constraint.Name(tableName))
	sql = append(sql, " PRIMARY KEY ("...)

	for i, column := range constraint.ColumnNames {
		if i > 0 {
			sql = append(sql, ", "...)
		}
		sql = fmter.AppendIdent(sql, string(column))
	}

	sql = append(sql, ")"...)

	return sql
}

func (constraint *PrimaryKeyConstraint) ToDropSQL(fmter SQLFormatter, tableName TableName) []byte {
	sql := []byte{}

	sql = append(sql, "ALTER TABLE "...)
	sql = fmter.AppendIdent(sql, string(tableName))
	sql = append(sql, " DROP CONSTRAINT IF EXISTS "...)
	sql = fmter.AppendIdent(sql, constraint.Name(tableName))

	return sql
}

type ForeignKeyConstraint struct {
	ReferencingColumnName ColumnName
	ReferencedTableName   TableName
	ReferencedColumnName  ColumnName
	name                  string
}

func (constraint *ForeignKeyConstraint) Name(tableName TableName) string {
	if constraint.name != "" {
		return constraint.name
	}

	var b strings.Builder
	b.WriteString(ConstraintTypeForeignKey.String())
	b.WriteString("_")
	b.WriteString(string(tableName))
	b.WriteString("_")
	b.WriteString(string(constraint.ReferencingColumnName))
	return b.String()
}

func (constraint *ForeignKeyConstraint) Named(name string) Constraint {
	return &ForeignKeyConstraint{
		ReferencingColumnName: constraint.ReferencingColumnName,
		ReferencedTableName:   constraint.ReferencedTableName,
		ReferencedColumnName:  constraint.ReferencedColumnName,
		name:                  name,
	}
}

func (constraint *ForeignKeyConstraint) Type() ConstraintType {
	return ConstraintTypeForeignKey
}

func (constraint *ForeignKeyConstraint) Columns() []ColumnName {
	return []ColumnName{constraint.ReferencingColumnName}
}

func (constraint *ForeignKeyConstraint) Equals(other Constraint) bool {
	if other.Type() != ConstraintTypeForeignKey {
		return false
	}

	otherForeignKeyConstraint, ok := other.(*ForeignKeyConstraint)
	if !ok {
		return false
	}

	return constraint.ReferencingColumnName == otherForeignKeyConstraint.ReferencingColumnName &&
		constraint.ReferencedTableName == otherForeignKeyConstraint.ReferencedTableName &&
		constraint.ReferencedColumnName == otherForeignKeyConstraint.ReferencedColumnName
}

func (constraint *ForeignKeyConstraint) ToDefinitionSQL(fmter SQLFormatter, tableName TableName) []byte {
	sql := []byte{}

	sql = append(sql, "CONSTRAINT "...)
	sql = fmter.AppendIdent(sql, constraint.Name(tableName))
	sql = append(sql, " FOREIGN KEY ("...)

	sql = fmter.AppendIdent(sql, string(constraint.ReferencingColumnName))
	sql = append(sql, ") REFERENCES "...)
	sql = fmter.AppendIdent(sql, string(constraint.ReferencedTableName))
	sql = append(sql, " ("...)
	sql = fmter.AppendIdent(sql, string(constraint.ReferencedColumnName))
	sql = append(sql, ")"...)

	return sql
}

func (constraint *ForeignKeyConstraint) ToDropSQL(fmter SQLFormatter, tableName TableName) []byte {
	sql := []byte{}

	sql = append(sql, "ALTER TABLE "...)
	sql = fmter.AppendIdent(sql, string(tableName))
	sql = append(sql, " DROP CONSTRAINT IF EXISTS "...)
	sql = fmter.AppendIdent(sql, constraint.Name(tableName))

	return sql
}

// Do not use this constraint type. Instead create an index with the `UniqueIndex` type.
// The main difference between a Unique Index and a Unique Constraint is mostly semantic, with a constraint focusing more on data integrity, while an index focuses on performance.
// We choose to create unique indices because of sqlite. Dropping a unique index is directly supported whilst dropping a unique constraint requires a recreation of the table with the constraint removed.
type UniqueConstraint struct {
	ColumnNames []ColumnName
	name        string
}

func (constraint *UniqueConstraint) Name(tableName TableName) string {
	if constraint.name != "" {
		return constraint.name
	}

	var b strings.Builder
	b.WriteString(ConstraintTypeUnique.String())
	b.WriteString("_")
	b.WriteString(string(tableName))
	b.WriteString("_")
	for i, column := range constraint.ColumnNames {
		if i > 0 {
			b.WriteString("_")
		}
		b.WriteString(string(column))
	}
	return b.String()
}

func (constraint *UniqueConstraint) Named(name string) Constraint {
	copyOfColumnNames := make([]ColumnName, len(constraint.ColumnNames))
	copy(copyOfColumnNames, constraint.ColumnNames)

	return &UniqueConstraint{
		ColumnNames: copyOfColumnNames,
		name:        name,
	}
}

func (constraint *UniqueConstraint) Type() ConstraintType {
	return ConstraintTypeUnique
}

func (constraint *UniqueConstraint) Columns() []ColumnName {
	return constraint.ColumnNames
}

func (constraint *UniqueConstraint) Equals(other Constraint) bool {
	if other.Type() != ConstraintTypeUnique {
		return false
	}

	foundColumns := make(map[ColumnName]bool)
	for _, column := range constraint.ColumnNames {
		foundColumns[column] = true
	}

	for _, column := range other.Columns() {
		if !foundColumns[column] {
			return false
		}
	}

	return true
}

func (constraint *UniqueConstraint) ToIndex(tableName TableName) *UniqueIndex {
	copyOfColumnNames := make([]ColumnName, len(constraint.ColumnNames))
	copy(copyOfColumnNames, constraint.ColumnNames)

	return &UniqueIndex{
		TableName:   tableName,
		ColumnNames: copyOfColumnNames,
	}
}

func (constraint *UniqueConstraint) ToDefinitionSQL(fmter SQLFormatter, tableName TableName) []byte {
	sql := []byte{}

	sql = append(sql, "CONSTRAINT "...)
	sql = fmter.AppendIdent(sql, constraint.Name(tableName))
	sql = append(sql, " UNIQUE ("...)

	for i, column := range constraint.ColumnNames {
		if i > 0 {
			sql = append(sql, ", "...)
		}
		sql = fmter.AppendIdent(sql, string(column))
	}

	sql = append(sql, ")"...)

	return sql
}

func (constraint *UniqueConstraint) ToDropSQL(fmter SQLFormatter, tableName TableName) []byte {
	sql := []byte{}

	sql = append(sql, "ALTER TABLE "...)
	sql = fmter.AppendIdent(sql, string(tableName))
	sql = append(sql, " DROP CONSTRAINT IF EXISTS "...)
	sql = fmter.AppendIdent(sql, constraint.Name(tableName))

	return sql
}
