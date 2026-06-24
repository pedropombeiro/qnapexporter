// Package tagextractor extracts tags from notification text, optionally
// rewriting the annotation in the process.
package tagextractor

// TagExtractor extracts tags from an annotation, returning the (possibly
// modified) annotation along with the extracted tags.
type TagExtractor interface {
	// Extract takes an annotation and returns a modified annotation and an array of extracted tags
	Extract(annotation string) (string, []string)
}

type noOpTagExtractor struct {
}

// NewNoOpTagExtractor returns a TagExtractor that leaves the annotation
// unchanged and extracts no tags.
func NewNoOpTagExtractor() TagExtractor {
	return new(noOpTagExtractor)
}

func (c *noOpTagExtractor) Extract(annotation string) (string, []string) {
	return annotation, nil
}
