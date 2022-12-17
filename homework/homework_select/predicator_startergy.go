package orm

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

	return nil
}

func (p *Processor[T]) Where() error {
	p.selector.sb.WriteString(" WHERE ")
	pre := p.selector.where[0]
	for i := 1; i < len(p.selector.where); i++ {
		pre = pre.And(p.selector.where[i])
	}
	if err := p.selector.buildExpression(pre); err != nil {
		return err
	}
	return nil
}

func (p *Processor[T]) OrderBy() {

}

func (p *Processor[T]) GroupBy() {

}

func (p *Processor[T]) Having() {

}
