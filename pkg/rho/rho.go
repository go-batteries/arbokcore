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
