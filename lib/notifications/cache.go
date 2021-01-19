package notifications

type AnnotationCache interface {
	Match(annotation string) (int, error)
}

type noOpAnnotationCache struct {
}

func NewAnnotationCache() AnnotationCache {
	return new(noOpAnnotationCache)
}

func (c *noOpAnnotationCache) Match(annotation string) (int, error) {
	return -1, nil
}
