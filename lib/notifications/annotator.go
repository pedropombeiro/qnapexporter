package notifications

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Annotator interface {
	Post(annotation string) (int, error)
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type regionMatchingAnnotator struct {
	grafanaURL       string
	grafanaAuthToken string
	tags             []string
	cache            AnnotationCache
	client           httpClient
	logger           *log.Logger
}

func NewAnnotator(
	grafanaURL, grafanaAuthToken string,
	tags []string,
	cache AnnotationCache,
	c httpClient,
	logger *log.Logger,
) Annotator {
	return &regionMatchingAnnotator{
		grafanaURL:       grafanaURL,
		grafanaAuthToken: grafanaAuthToken,
		tags:             tags,
		cache:            cache,
		client:           c,
		logger:           logger,
	}
}

func (a *regionMatchingAnnotator) Post(annotation string) (int, error) {
	tags := make([]string, 0, len(a.tags))
	for _, t := range a.tags {
		tags = append(tags, `"`+t+`"`)
	}

	url := fmt.Sprintf("%s/api/annotations", a.grafanaURL)
	body := fmt.Sprintf(`{"tags":[%s],"text":%q}`, strings.Join(tags, ","), annotation)

	reqType := "POST"
	id, err := a.cache.Match(annotation)
	if err == nil && id != -1 {
		reqType = "PATCH"
		body = fmt.Sprintf(`{"timeEnd":%d}`, time.Now().UnixNano()/1000)
		url = fmt.Sprintf("%s/%d", url, id)
	}

	bodyReader := strings.NewReader(body)
	req, err := http.NewRequest(reqType, url, bodyReader)
	if err != nil {
		a.logger.Printf("Error creating Grafana annotation request: %v\n", err)
		return -1, err
	}

	req.Header.Set("Content-Type", "application/json")
	if a.grafanaAuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.grafanaAuthToken))
	}

	resp, err := a.client.Do(req)
	if err == nil {
		a.logger.Printf("Created Grafana annotation at %s: %s\n", url, resp.Status)
	} else {
		a.logger.Printf("Error creating Grafana annotation at %s: %v\n", url, err)
	}

	return -1, err
}
