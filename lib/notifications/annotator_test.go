package notifications

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAnnotator(t *testing.T) {
	a := NewAnnotator(
		"",
		"",
		[]string{},
		new(MockAnnotationCache),
		new(mockHttpClient),
		log.New(ioutil.Discard, "", 0),
	)

	require.NotNil(t, a)
	assert.IsType(t, &regionMatchingAnnotator{}, a)
}

func TestPostAnnotation(t *testing.T) {
	testCases := map[string]struct {
		testURL         string
		testAuthToken   string
		tags            []string
		setupCacheMock  func(c *MockAnnotationCache)
		setupClientMock func(c *mockHttpClient)
		notification    string
		expectedID      int
		expectedErr     error
	}{
		"success": {
			testURL:       "http://grafana.example.com",
			testAuthToken: "token1",
			tags:          []string{"tag1", "tag2"},
			setupCacheMock: func(c *MockAnnotationCache) {
				c.On("Match", "test notification").
					Once().
					Return(-1, nil)
			},
			setupClientMock: func(c *mockHttpClient) {
				c.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return assert.Equal(t, "POST", req.Method) &&
						assert.Equal(t, "grafana.example.com", req.Host) &&
						assert.Equal(t, "application/json", req.Header.Get("Content-Type")) &&
						assert.Equal(t, "Bearer token1", req.Header.Get("Authorization")) &&
						assert.Equal(t, `{"tags":["tag1","tag2"],"text":"test notification"}`, readAll(req.Body))
				})).
					Once().
					Return(responseWithBody(`{"id": 1}`), nil)
			},
			notification: "test notification",
			expectedID:   1,
			expectedErr:  nil,
		},
		"patch existing annotation": {
			testURL:       "http://grafana.com",
			testAuthToken: "token2",
			tags:          []string{"tag1"},
			setupCacheMock: func(c *MockAnnotationCache) {
				c.On("Match", "patch notification").
					Once().
					Return(98, nil)
			},
			setupClientMock: func(c *mockHttpClient) {
				c.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					body := readAll(req.Body)
					return assert.Equal(t, "PATCH", req.Method) &&
						assert.Equal(t, "grafana.com", req.Host) &&
						assert.Equal(t, "/api/annotations/98", req.URL.Path) &&
						assert.Equal(t, "application/json", req.Header.Get("Content-Type")) &&
						assert.Equal(t, "Bearer token2", req.Header.Get("Authorization")) &&
						assert.Contains(t, body, `"tags":["tag1"],`) &&
						assert.Contains(t, body, `"timeEnd":`) &&
						assert.Contains(t, body, `text":"patch notification"}`)
				})).
					Once().
					Return(responseWithBody(`{"id": 98}`), nil)
			},
			notification: "patch notification",
			expectedID:   98,
			expectedErr:  nil,
		},
		"HTTP client returns error": {
			testURL:       "http://grafana.com",
			testAuthToken: "token2",
			tags:          []string{"tag1"},
			setupCacheMock: func(c *MockAnnotationCache) {
				c.On("Match", "test notification").
					Once().
					Return(-1, nil)
			},
			setupClientMock: func(c *mockHttpClient) {
				c.On("Do", mock.Anything).
					Once().
					Return(nil, assert.AnError)
			},
			notification: "test notification",
			expectedID:   -1,
			expectedErr:  assert.AnError,
		},
		"grafana returns error": {
			testURL:       "http://grafana.com",
			testAuthToken: "token2",
			tags:          []string{"tag1"},
			setupCacheMock: func(c *MockAnnotationCache) {
				c.On("Match", "test notification").
					Once().
					Return(-1, nil)
			},
			setupClientMock: func(c *mockHttpClient) {
				c.On("Do", mock.Anything).
					Once().
					Return(&http.Response{StatusCode: 404, Status: "Not found"}, nil)
			},
			notification: "test notification",
			expectedID:   -1,
			expectedErr:  fmt.Errorf("call to %s failed with HTTP %d %q", "http://grafana.com/api/annotations", 404, "Not found"),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			cacheMock := &MockAnnotationCache{}
			clientMock := &mockHttpClient{}
			tc.setupCacheMock(cacheMock)
			tc.setupClientMock(clientMock)

			a := NewAnnotator(
				tc.testURL,
				tc.testAuthToken,
				tc.tags,
				cacheMock,
				clientMock,
				log.New(ioutil.Discard, "", 0),
			)

			id, err := a.Post(tc.notification)

			if tc.expectedErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, id)
			} else {
				assert.Equal(t, tc.expectedErr, err)
			}
		})
	}
}

func readAll(r io.Reader) string {
	s, err := ioutil.ReadAll(r)
	if err != nil {
		return err.Error()
	}

	return string(s)
}

func responseWithBody(body string) *http.Response {
	return &http.Response{Body: ioutil.NopCloser(strings.NewReader(body))}
}
