package tagextractor

import "strings"

type notificationCenterTagExtractor struct {
}

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
