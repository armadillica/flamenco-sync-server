package main

import (
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"

	check "gopkg.in/check.v1"
	"gopkg.in/jarcoal/httpmock.v1"
)

type HTTPHandlerTestSuite struct {
	fss *syncServer
}

var _ = check.Suite(&HTTPHandlerTestSuite{})

func (s *HTTPHandlerTestSuite) SetUpTest(c *check.C) {
	httpmock.Activate()
	s.fss = createSyncServer()
}

func (s *HTTPHandlerTestSuite) TearDownTest(c *check.C) {
	httpmock.DeactivateAndReset()
}

func (s *HTTPHandlerTestSuite) TestGET(t *check.C) {
	respRec := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/etc/passwd", nil)
	s.fss.ServeHTTP(respRec, request)

	assert.Equal(t, http.StatusMethodNotAllowed, respRec.Code)
}

func (s *HTTPHandlerTestSuite) TestRsyncNoWebsocket(t *check.C) {
	respRec := httptest.NewRecorder()
	request, _ := http.NewRequest("RSYNC", "/etc/passwd", nil)
	s.fss.ServeHTTP(respRec, request)

	assert.Equal(t, http.StatusNotImplemented, respRec.Code)
}

func (s *HTTPHandlerTestSuite) TestRsyncWebsocketHappy(t *check.C) {
	respRec := httptest.NewRecorder()
	request, _ := http.NewRequest("RSYNC", "/etc/passwd", nil)
	request.Header.Set("Upgrade", "websocket")
	s.fss.ServeHTTP(respRec, request)

	assert.Equal(t, http.StatusOK, respRec.Code)
}
