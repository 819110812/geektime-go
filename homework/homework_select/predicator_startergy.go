package orm

import "fmt"

type PredicateProcessor[T any] interface {
	process(s *Selector[T])
}

type Processor[T any] struct {
	selector *Selector[T]
}

func NewPredicateProcess[T any](s *Selector[T]) *Processor[T] {
	return &Processor[T]{
		selector: s,
	}
}

func (p *Processor[T]) process() error {
	if len(p.selector.where) > 0 {
		err := p.Where()
		if err != nil {
			return err
		}
	}

	if len(p.selector.groupBy) > 0 {
		err := p.GroupBy()
		if err != nil {
			return err
		}
	}

	if len(p.selector.orderBy) > 0 {
		err := p.OrderBy()
		if err != nil {
			return err
		}
	}

	if len(p.selector.having) > 0 {
		err := p.Having()
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Processor[T]) Where() error {
	p.selector.sb.WriteString(" WHERE ")
	pre := p.selector.where[0]
	for i := 1; i < len(p.selector.where); i++ {
		pre = pre.And(p.selector.where[i])
	}
	//fmt.Println(pre.le)
	if err := p.selector.buildExpression(pre); err != nil {
		return err
	}
	return nil
}

func (p *Processor[T]) OrderBy() error {
	p.selector.sb.WriteString(" ORDER BY ")
	for i, order := range p.selector.orderBy {
		if i > 0 {
			p.selector.sb.WriteByte(',')
		}
		err := p.selector.buildColumn(order.Column)
		if err != nil {
			return err
		}
		p.selector.sb.WriteByte(' ')
		if order.Asc {
			p.selector.sb.WriteString("ASC")
		} else {
			p.selector.sb.WriteString("DESC")
		}
	}

	return nil
}

func (p *Processor[T]) GroupBy() error {
	p.selector.sb.WriteString(" GROUP BY ")
	fmt.Println(p.selector.groupBy)
	for i, v := range p.selector.groupBy {
		if i > 0 {
			p.selector.sb.WriteByte(',')
		}
		v.alias = ""
		err := p.selector.buildColumn(v)
		if err != nil {
			return err
		}
	}

	return nil

}

func (p *Processor[T]) Having() error {
	return nil
}
