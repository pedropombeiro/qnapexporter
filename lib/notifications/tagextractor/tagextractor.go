package tagextractor

type TagExtractor interface {
	// Extract takes an annotation and returns a modified annotation and an array of extracted tags
	Extract(annotation string) (string, []string)
}

type noOpTagExtractor struct {
}

func NewNoOpTagExtractor() TagExtractor {
	return new(noOpTagExtractor)
}

func (c *noOpTagExtractor) Extract(annotation string) (string, []string) {
	return annotation, nil
}
