package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pedropombeiro/qnapexporter/lib/notifications/tagextractor"
)

type Annotator interface {
	Post(annotation string, time time.Time) (int, error)
}

type grafanaAnnotation struct {
	Id      int      `json:"id,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Time    int64    `json:"time,omitempty"`
	TimeEnd int64    `json:"timeEnd,omitempty"`
	Text    string   `json:"text,omitempty"`
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewSimpleAnnotator(
	grafanaURL, grafanaAuthToken string,
	tags []string,
	c httpClient,
	logger *log.Logger,
) Annotator {
	if len(tags) == 1 && tags[0] == "" {
		tags = nil
	}

	return &regionMatchingAnnotator{
		grafanaURL:       grafanaURL,
		grafanaAuthToken: grafanaAuthToken,
		tags:             tags,
		tagExtractor:     tagextractor.NewNoOpTagExtractor(),
		cache:            NewNoOpRegionMatcher(),
		client:           c,
		logger:           logger,
	}
}

type regionMatchingAnnotator struct {
	grafanaURL       string
	grafanaAuthToken string
	tags             []string
	tagExtractor     tagextractor.TagExtractor
	cache            RegionMatcher
	client           httpClient
	logger           *log.Logger
}

func NewRegionMatchingAnnotator(
	grafanaURL, grafanaAuthToken string,
	tags []string,
	tagExtractor tagextractor.TagExtractor,
	cache RegionMatcher,
	c httpClient,
	logger *log.Logger,
) Annotator {
	if len(tags) == 1 && tags[0] == "" {
		tags = nil
	}

	return &regionMatchingAnnotator{
		grafanaURL:       grafanaURL,
		grafanaAuthToken: grafanaAuthToken,
		tags:             tags,
		tagExtractor:     tagExtractor,
		cache:            cache,
		client:           c,
		logger:           logger,
	}
}

func (a *regionMatchingAnnotator) Post(annotation string, time time.Time) (int, error) {
	url := fmt.Sprintf("%s/api/annotations", a.grafanaURL)
	trimmedAnnotation, annotationTags := a.tagExtractor.Extract(annotation)
	ga := grafanaAnnotation{
		Text: trimmedAnnotation,
		Tags: mergeTags(a.tags, annotationTags),
		Time: time.UnixNano() / 1000000,
	}
	id := a.cache.Match(annotation)

	reqType := "POST"
	if id != -1 {
		reqType = "PATCH"
		ga.Time = 0
		ga.TimeEnd = time.UnixNano() / 1000000
		url = fmt.Sprintf("%s/%d", url, id)
	}

	jsonBytes, err := json.Marshal(ga)
	if err != nil {
		a.logger.Printf("Error marshalling Grafana annotation: %v\n", err)
		return -1, err
	}
	bodyReader := bytes.NewReader(jsonBytes)
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
		if resp.StatusCode < 300 {
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				return -1, fmt.Errorf("reading response body: %w", readErr)
			}

			var response struct {
				Id      int    `json:"id"`
				Message string `json:"message"`
			}
			err = json.Unmarshal(body, &response)
			if err != nil {
				return -1, fmt.Errorf("unmarshaling response body: %w", err)
			}

			if id == -1 {
				a.cache.Add(response.Id, annotation)
			}

			a.logger.Printf("%s (status: %q), ID: %d\n", response.Message, resp.Status, response.Id)
			return response.Id, nil
		}

		a.logger.Printf("Error creating Grafana annotation at %s: HTTP %d %q\n", url, resp.StatusCode, resp.Status)
		err = fmt.Errorf("call to %s failed with HTTP %d %q", url, resp.StatusCode, resp.Status)
	} else {
		a.logger.Printf("Error creating Grafana annotation at %s: %v\n", url, err)
	}

	return -1, err
}

func mergeTags(t1 []string, t2 []string) []string {
	keys := make(map[string]bool)
	list := make([]string, 0, len(t1)+len(t2))

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range append(t1, t2...) {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
