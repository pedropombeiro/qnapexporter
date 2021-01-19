package notifications

import "regexp"

type AnnotationCache interface {
	Add(id int, annotation string)
	Match(annotation string) int
}

type noOpAnnotationCache struct {
}

func NewNoOpAnnotationCache() AnnotationCache {
	return new(noOpAnnotationCache)
}

func (c *noOpAnnotationCache) Add(id int, annotation string) {
}

func (c *noOpAnnotationCache) Match(annotation string) int {
	return -1
}

// replacementRules describes a regular expression to match the end event, and a substitution to convert it to the start event
type replacementRule struct {
	re           *regexp.Regexp
	substitution string
}

var rules = []replacementRule{
	{re: regexp.MustCompile(`\[Malware Remover\] Scan completed\.`), substitution: `[Malware Remover] Started scanning.`},
	{re: regexp.MustCompile(`\[Storage & Snapshots\] Finished(\s.*)`), substitution: `[Storage & Snapshots] Started$1`},
	{re: regexp.MustCompile(`\[Firmware Update\] Started updating firmware`), substitution: `[Firmware Update] Started downloading firmware`},
	{re: regexp.MustCompile(`\[Firmware Update\] Updated system\.`), substitution: `[Firmware Update] Started updating firmware.`},
	{re: regexp.MustCompile(`\[Disk S\.M\.A\.R\.T\.\] (.+) Rapid Test result:.*`), substitution: "[Disk S.M.A.R.T.] $1 Rapid Test started."},
	{re: regexp.MustCompile(`\[Antivirus\] Completed scan job ("[^"]+").+`), substitution: `[Antivirus] Started scan job $1.`},
	{re: regexp.MustCompile(`\[SortMyQPKGs\] ('.+') completed`), substitution: `[SortMyQPKGs] $1 requested`},
	{re: regexp.MustCompile(`\[RunLast\] end ("[^"]+") scripts`), substitution: `[RunLast] begin $1 scripts ...`},
	{re: regexp.MustCompile(`\[SecurityCounselor\] Finished`), substitution: "[SecurityCounselor] Started"},
}

type cacheEntry struct {
	id         int
	annotation string
}

type matcherAnnotationCache struct {
	cacheSize int
	cache     []cacheEntry
}

func NewMatcherAnnotationCache(cacheSize int) AnnotationCache {
	return &matcherAnnotationCache{
		cacheSize: cacheSize,
	}
}

func (c *matcherAnnotationCache) Add(id int, annotation string) {
	c.cache = append(c.cache, cacheEntry{
		id:         id,
		annotation: annotation,
	})
	if len(c.cache) > c.cacheSize {
		c.cache = c.cache[1:]
	}
}

func (c *matcherAnnotationCache) Match(annotation string) (id int) {
	idx := -1
	defer func() {
		if idx >= 0 {
			id = c.cache[idx].id

			// Delete the cache entry
			c.cache = append(c.cache[:idx], c.cache[idx+1:]...)
		}
	}()

	for _, r := range rules {
		previousAnnotation := r.re.ReplaceAllString(annotation, r.substitution)
		if previousAnnotation != annotation {
			idx = c.findIndex(previousAnnotation)
			if idx >= 0 {
				return
			}
		}
	}

	return -1
}

func (c *matcherAnnotationCache) findIndex(annotation string) int {
	for idx, entry := range c.cache {
		if entry.annotation == annotation {
			return idx
		}
	}

	return -1
}
