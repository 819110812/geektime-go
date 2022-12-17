package orm

import (
	"fmt"
	"homework/homework_select/internal/errs"
	"homework/homework_select/model"
	"log"
	"strings"
)

// Selector 用于构造 SELECT 语句
type Selector[T any] struct {
	db      *DB
	sb      strings.Builder
	args    []any
	table   string
	where   []Predicate
	raw     string
	model   *model.Model
	columns []Selectable
	having  []Predicate
	groupBy []Column
	orderBy []OrderBy
	offset  int
	limit   int
	alias   []string
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	s.columns = cols
	return s
}

// From 指定表名，如果是空字符串，那么将会使用默认表名
func (s *Selector[T]) From(tbl string) *Selector[T] {
	s.table = tbl
	return s
}

func (s *Selector[T]) Build() (*Query, error) {
	//s.sb = strings.Builder{}
	if s.model == nil {
		var err error
		s.model, err = s.db.registry.Get(new(T))
		if err != nil {
			return nil, err
		}
	}
	s.sb.WriteString("SELECT ")

	if err := s.buildColumns(); err != nil {
		return nil, err
	}

	s.sb.WriteString(" FROM ")

	if s.table == "" {
		var t T
		mod, err := s.db.registry.Get(&t)
		if err != nil {
			log.Println("failed to get model ", err.Error())
			return nil, err
		}
		s.sb.WriteByte('`')
		s.sb.WriteString(mod.TableName)
		s.sb.WriteByte('`')
	} else {
		s.sb.WriteString(s.table)
	}

	processor := NewPredicateProcess(s)

	err := processor.process()
	if err != nil {
		return nil, err
	}

	s.sb.WriteString(";")
	return &Query{
		SQL:  s.sb.String(),
		Args: s.args,
	}, nil
}

func (s *Selector[T]) buildColumns() error {
	if len(s.columns) == 0 {
		s.sb.WriteByte('*')
	}
	for i, c := range s.columns {
		if i > 0 {
			s.sb.WriteByte(',')
		}
		switch val := c.(type) {
		case Column:
			if err := s.buildColumn(val); err != nil {
				return err
			}
		case RawExpr:
			s.sb.WriteString(val.raw)
			if val.args != nil {
				s.args = append(s.args, val.args...)
			}
		case Aggregate:
			if err := s.buildAggregate(val, true); err != nil {
				return err
			}
		default:
			fmt.Println("unsupported column type", val)
			return errs.NewErrUnsupportedSelectable(c)
		}
	}
	return nil
}

func (s *Selector[T]) buildExpression(e Expression) error {
	if e == nil {
		return nil
	}
	switch exp := e.(type) {
	case Column:
		//s.sb.WriteByte('`')
		//s.sb.WriteString(exp.name)
		//s.sb.WriteByte('`')
		if len(s.where) > 0 {
			exp.alias = ""
		}
		err := s.buildColumn(exp)
		if err != nil {
			return err
		}
	case value:
		s.sb.WriteByte('?')
		s.args = append(s.args, exp.val)
	case Predicate:
		_, lp := exp.left.(Predicate)
		if lp {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.left); err != nil {
			return err
		}

		if lp {
			s.sb.WriteByte(')')
		}

		if exp.right != nil {
			s.sb.WriteByte(' ')
			s.sb.WriteString(exp.op.String())
			s.sb.WriteByte(' ')
		}

		_, rp := exp.right.(Predicate)
		if rp {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.right); err != nil {
			return err
		}
		if rp {
			s.sb.WriteByte(')')
		}
	case RawExpr:
		s.sb.WriteString(exp.raw)
		if exp.args != nil {
			s.args = append(s.args, exp.args...)
		}
	default:
		fmt.Println("unsupported expression", e)
		return fmt.Errorf("orm: 不支持的表达式 %v", exp)
	}
	return nil
}

// Where 用于构造 WHERE 查询条件。如果 ps 长度为 0，那么不会构造 WHERE 部分
func (s *Selector[T]) Where(ps ...Predicate) *Selector[T] {
	s.where = ps
	return s
}

// GroupBy 设置 group by 子句
func (s *Selector[T]) GroupBy(cols ...Column) *Selector[T] {
	s.groupBy = cols
	return s
}

func (s *Selector[T]) Having(ps ...Predicate) *Selector[T] {
	s.having = ps
	return s
}

func (s *Selector[T]) Offset(offset int) *Selector[T] {
	s.offset = offset
	return s
}

func (s *Selector[T]) Limit(limit int) *Selector[T] {
	s.limit = limit
	return s
}

func (s *Selector[T]) OrderBy(orderBys ...OrderBy) *Selector[T] {
	s.orderBy = orderBys
	return s
}

func (s *Selector[T]) buildColumn(c Column) error {
	fd, ok := s.model.FieldMap[c.name]
	if !ok {
		return errs.NewErrUnknownField(c.name)
	}
	s.sb.WriteByte('`')
	s.sb.WriteString(fd.ColName)
	s.sb.WriteByte('`')
	if c.alias != "" {
		s.sb.WriteString(" AS `")
		s.sb.WriteString(c.alias)
		s.sb.WriteByte('`')
	}
	return nil
}

func (s *Selector[T]) buildAggregate(val Aggregate, useAlias bool) error {
	s.sb.WriteString(val.fn)
	s.sb.WriteString("(`")
	fd, ok := s.model.FieldMap[val.arg]
	if !ok {
		return errs.NewErrUnknownField(val.arg)
	}
	s.sb.WriteString(fd.ColName)
	s.sb.WriteString("`)")
	if useAlias {
		s.buildAs(val.alias)
	}
	return nil
}

func (s *Selector[T]) buildAs(alias string) {
	if alias != "" {
		s.sb.WriteString(" AS ")
		s.sb.WriteByte('`')
		s.sb.WriteString(alias)
		s.sb.WriteByte('`')
	}
}

func NewSelector[T any](db *DB) *Selector[T] {
	return &Selector[T]{
		db: db,
	}
}

type Selectable interface {
	selectable()
}

type OrderBy struct {
	Column Column
	Asc    bool
}

func Asc(col string) OrderBy {
	return OrderBy{Column: Column{name: col}, Asc: true}
}

func Desc(col string) OrderBy {
	return OrderBy{Column: Column{name: col}, Asc: false}
}
