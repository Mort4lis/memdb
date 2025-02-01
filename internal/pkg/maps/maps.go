package maps

func Reverse[K, V comparable](m map[K]V) map[V]K {
	cp := make(map[V]K, len(m))
	for k, v := range m {
		cp[v] = k
	}
	return cp
}
