package tagextractor

import "strings"

type notificationCenterTagExtractor struct {
}

// NewNotificationCenterTagExtractor returns a TagExtractor that parses QNAP
// Notification Center messages, extracting their leading bracketed component as
// a tag.
func NewNotificationCenterTagExtractor() TagExtractor {
	return new(notificationCenterTagExtractor)
}

func (c *notificationCenterTagExtractor) Extract(annotation string) (string, []string) {
	var tags []string

	for annotation[0] == '[' {
		endIdx := strings.Index(annotation[1:], "] ")
		if endIdx == -1 {
			break
		}

		endIdx++
		tags = append(tags, annotation[1:endIdx])
		annotation = annotation[endIdx+2:]
	}

	return annotation, tags
}
