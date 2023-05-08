package util

type Pool[T any] struct {
	Items    chan T
	Factory  func() T
	Close    func(T)
	AfterPut func(T)
}

func (p *Pool[T]) Get() T {
	var item T
	select {
	case item = <-p.Items:
	default:
		item = p.Factory()
	}
	return item
}

func (p *Pool[T]) Put(item T) error {
	if p.Items == nil {
		// pool is closed, close passed client
		p.Close(item)
		return nil
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case p.Items <- item:
		if p.AfterPut != nil {
			p.AfterPut(item)
		}
		return nil
	default:
		// pool is full, close passed client
		p.Close(item)
		return nil
	}
}

func (p *Pool[T]) Len() int {
	return len(p.Items)
}

func (p *Pool[T]) Done() {
	close(p.Items)
	for i := range p.Items {
		p.Close(i)
	}
}
