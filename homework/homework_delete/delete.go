package homework_delete

import (
	"reflect"
	"strings"
)

type Deleter[T any] struct {
	sb        *strings.Builder
	tableName string
	where     []Predicate
	args      []any
}

func (d *Deleter[T]) Build() (*Query, error) {
	d.sb = &strings.Builder{}
	d.sb.WriteString("DELETE FROM ")
	if d.tableName == "" {
		d.sb.WriteByte('`')
		var table T
		tableName := reflect.TypeOf(table)
		d.sb.WriteString(tableName.Name())
		d.sb.WriteByte('`')
	} else {
		d.sb.WriteString(d.tableName)
	}

	if len(d.where) > 0 {
		d.sb.WriteString(" WHERE ")
		p := d.where[0]
		for i := 1; i < len(d.where); i++ {
			p = p.And(d.where[i])
		}
		if err := d.BuildExpression(p); err != nil {
			return nil, err
		}

	}

	d.sb.WriteByte(';')
	return &Query{
		SQL:  d.sb.String(),
		Args: d.args,
	}, nil
}

// From accepts model definition
func (d *Deleter[T]) From(table string) *Deleter[T] {
	d.tableName = table
	return d
}

// BuildExpression build expr for sql
func (d *Deleter[T]) BuildExpression(expression Expression) error {
	if expression == nil {
		return nil
	}
	switch typ := expression.(type) {
	case Column:
		d.sb.WriteByte('`')
		d.sb.WriteString(typ.name)
		d.sb.WriteByte('`')
	case value:
		d.sb.WriteByte('?')
		d.args = append(d.args, typ.val)
	case Predicate:
		_, left := typ.left.(Predicate)
		if left {
			d.sb.WriteByte('(')
		}

		if err := d.BuildExpression(typ.left); err != nil {
			return err
		}

		_, right := typ.right.(Predicate)
		if right {
			d.sb.WriteByte(')')
		}

		d.sb.WriteByte(' ')
		d.sb.WriteString(typ.op.String())
		d.sb.WriteByte(' ')

		if err := d.BuildExpression(typ.right); err != nil {
			return err
		}
	}
	return nil
}

// Where accepts predicates
func (d *Deleter[T]) Where(predicates ...Predicate) *Deleter[T] {
	d.where = predicates
	return d
}
