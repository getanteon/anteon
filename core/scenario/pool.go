package scenario

type Pool[T any] struct {
	items   chan T
	factory func() T
	close   func(T)
}

func (p *Pool[T]) Get() T {
	var item T
	select {
	case item = <-p.items:
	default:
		item = p.factory()
	}
	return item
}

func (p *Pool[T]) Put(item T) error {
	if p.items == nil {
		// pool is closed, close passed client
		p.close(item)
		return nil
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case p.items <- item:
		return nil
	default:
		// pool is full, close passed client
		p.close(item)
		return nil
	}
}

func (p *Pool[T]) Len() int {
	return len(p.items)
}

func (p *Pool[T]) Done() {
	close(p.items)
	for i := range p.items {
		p.close(i)
	}
}
