package notifications

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pedropombeiro/qnapexporter/lib/notifications/tagextractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAnnotator(t *testing.T) {
	a := NewRegionMatchingAnnotator(
		"",
		"",
		strings.Split("", ","),
		new(tagextractor.MockTagExtractor),
		new(MockRegionMatcher),
		new(mockHttpClient),
		log.New(io.Discard, "", 0),
	)

	require.NotNil(t, a)
	require.IsType(t, &regionMatchingAnnotator{}, a)
	c := a.(*regionMatchingAnnotator)
	assert.Empty(t, c.tags)
}

func TestPostAnnotation(t *testing.T) {
	testCases := map[string]struct {
		testURL               string
		testAuthToken         string
		tags                  []string
		setupTagExtractorMock func(m *tagextractor.MockTagExtractor)
		setupCacheMock        func(m *MockRegionMatcher)
		setupClientMock       func(m *mockHttpClient)
		notification          string
		expectedID            int
		expectedErr           error
	}{
		"success": {
			testURL:       "http://grafana.example.com",
			testAuthToken: "token1",
			tags:          []string{"tag1", "tag2"},
			setupTagExtractorMock: func(m *tagextractor.MockTagExtractor) {
				m.On("Extract", "test notification").
					Once().
					Return("test notification", nil)
			},
			setupCacheMock: func(m *MockRegionMatcher) {
				m.On("Match", "test notification").
					Once().
					Return(-1, nil)
				m.On("Add", 1, "test notification").
					Once()
			},
			setupClientMock: func(m *mockHttpClient) {
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return assert.Equal(t, "POST", req.Method) &&
						assert.Equal(t, "grafana.example.com", req.Host) &&
						assert.Equal(t, "application/json", req.Header.Get("Content-Type")) &&
						assert.Equal(t, "Bearer token1", req.Header.Get("Authorization")) &&
						assert.Equal(t, `{"tags":["tag1","tag2"],"time":1577880000000,"text":"test notification"}`, readBody(req))
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
			setupTagExtractorMock: func(m *tagextractor.MockTagExtractor) {
				m.On("Extract", "patch notification").
					Once().
					Return("patch notification", nil)
			},
			setupCacheMock: func(m *MockRegionMatcher) {
				m.On("Match", "patch notification").
					Once().
					Return(98, nil)
			},
			setupClientMock: func(m *mockHttpClient) {
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					body := readBody(req)
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
		"tags are extracted": {
			testURL:       "http://grafana.example.com",
			testAuthToken: "token1",
			tags:          []string{"tag1"},
			setupTagExtractorMock: func(m *tagextractor.MockTagExtractor) {
				m.On("Extract", "[tag2] test notification").
					Once().
					Return("test notification", []string{"tag2"})
			},
			setupCacheMock: func(m *MockRegionMatcher) {
				m.On("Match", "[tag2] test notification").
					Once().
					Return(-1, nil)
				m.On("Add", 1, "[tag2] test notification").
					Once()
			},
			setupClientMock: func(m *mockHttpClient) {
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return assert.Equal(t, "POST", req.Method) &&
						assert.Equal(t, "grafana.example.com", req.Host) &&
						assert.Equal(t, "application/json", req.Header.Get("Content-Type")) &&
						assert.Equal(t, "Bearer token1", req.Header.Get("Authorization")) &&
						assert.Equal(t, `{"tags":["tag1","tag2"],"time":1577880000000,"text":"test notification"}`, readBody(req))
				})).
					Once().
					Return(responseWithBody(`{"id": 1}`), nil)
			},
			notification: "[tag2] test notification",
			expectedID:   1,
			expectedErr:  nil,
		},
		"tags are extracted and deduplicated": {
			testURL:       "http://grafana.example.com",
			testAuthToken: "token1",
			tags:          []string{"tag1"},
			setupTagExtractorMock: func(m *tagextractor.MockTagExtractor) {
				m.On("Extract", "[tag1] [tag2] [tag3] test notification").
					Once().
					Return("test notification", []string{"tag1", "tag2", "tag3"})
			},
			setupCacheMock: func(m *MockRegionMatcher) {
				m.On("Match", "[tag1] [tag2] [tag3] test notification").
					Once().
					Return(-1, nil)
				m.On("Add", 1, "[tag1] [tag2] [tag3] test notification").
					Once()
			},
			setupClientMock: func(m *mockHttpClient) {
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return assert.Equal(t, "POST", req.Method) &&
						assert.Equal(t, "grafana.example.com", req.Host) &&
						assert.Equal(t, "application/json", req.Header.Get("Content-Type")) &&
						assert.Equal(t, "Bearer token1", req.Header.Get("Authorization")) &&
						assert.Equal(t, `{"tags":["tag1","tag2","tag3"],"time":1577880000000,"text":"test notification"}`, readBody(req))
				})).
					Once().
					Return(responseWithBody(`{"id": 1}`), nil)
			},
			notification: "[tag1] [tag2] [tag3] test notification",
			expectedID:   1,
			expectedErr:  nil,
		},
		"HTTP client returns error": {
			testURL:       "http://grafana.com",
			testAuthToken: "token2",
			tags:          []string{"tag1"},
			setupTagExtractorMock: func(m *tagextractor.MockTagExtractor) {
				m.On("Extract", "test notification").
					Once().
					Return("test notification", nil)
			},
			setupCacheMock: func(m *MockRegionMatcher) {
				m.On("Match", "test notification").
					Once().
					Return(-1, nil)
			},
			setupClientMock: func(m *mockHttpClient) {
				m.On("Do", mock.Anything).
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
			setupTagExtractorMock: func(m *tagextractor.MockTagExtractor) {
				m.On("Extract", "test notification").
					Once().
					Return("test notification", nil)
			},
			setupCacheMock: func(m *MockRegionMatcher) {
				m.On("Match", "test notification").
					Once().
					Return(-1, nil)
			},
			setupClientMock: func(m *mockHttpClient) {
				m.On("Do", mock.Anything).
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
			tagExtractorMock := new(tagextractor.MockTagExtractor)
			cacheMock := new(MockRegionMatcher)
			clientMock := new(mockHttpClient)
			defer func() {
				tagExtractorMock.AssertExpectations(t)
				cacheMock.AssertExpectations(t)
				clientMock.AssertExpectations(t)
			}()
			tc.setupTagExtractorMock(tagExtractorMock)
			tc.setupCacheMock(cacheMock)
			tc.setupClientMock(clientMock)

			a := NewRegionMatchingAnnotator(
				tc.testURL,
				tc.testAuthToken,
				tc.tags,
				tagExtractorMock,
				cacheMock,
				clientMock,
				log.New(io.Discard, "", 0),
			)

			id, err := a.Post(tc.notification, time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC))

			if tc.expectedErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, id)
			} else {
				assert.Equal(t, tc.expectedErr, err)
			}
		})
	}
}

func readBody(req *http.Request) string {
	body, err := req.GetBody()
	if err != nil {
		return err.Error()
	}

	b, err := io.ReadAll(body)
	if err != nil {
		return err.Error()
	}

	return string(b)
}

func responseWithBody(body string) *http.Response {
	return &http.Response{Body: io.NopCloser(strings.NewReader(body))}
}

func TestMergeTags(t *testing.T) {
	tags := mergeTags(nil, []string{"nas", "SecurityCounselor"})
	assert.Equal(t, []string{"nas", "SecurityCounselor"}, tags)

	tags = mergeTags([]string{"nas", "SecurityCounselor"}, nil)
	assert.Equal(t, []string{"nas", "SecurityCounselor"}, tags)

	tags = mergeTags([]string{"tag1", "tag2"}, []string{"tag2", "tag3"})
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, tags)
}
