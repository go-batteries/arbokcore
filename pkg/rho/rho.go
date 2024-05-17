package rho

type MapperFunc[E, V comparable] func(E, int) V

func Map[E, V comparable](arr []E, mapper MapperFunc[E, V]) []V {
	values := []V{}

	for i, el := range arr {
		v := mapper(el, i)
		values = append(values, v)
	}

	return values
}

type FilterFunc[E comparable] func(E, int) bool

func Filter[E comparable](arr []E, filter FilterFunc[E]) []E {
	values := []E{}

	for i, el := range arr {
		if ok := filter(el, i); ok {
			values = append(values, el)
		}
	}

	return values
}
