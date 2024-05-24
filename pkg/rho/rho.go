package rho

type MapperFunc[E, V any] func(E, int) V

func Map[E, V any](arr []E, mapper MapperFunc[E, V]) []V {
	values := []V{}

	for i, el := range arr {
		v := mapper(el, i)
		values = append(values, v)
	}

	return values
}

type FilterFunc[E any] func(E, int) bool

func Filter[E any](arr []E, filter FilterFunc[E]) []E {
	values := []E{}

	for i, el := range arr {
		if ok := filter(el, i); ok {
			values = append(values, el)
		}
	}

	return values
}
