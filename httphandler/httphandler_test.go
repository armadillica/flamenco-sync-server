package httphandler

import (
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"

	check "gopkg.in/check.v1"
	"gopkg.in/jarcoal/httpmock.v1"
)

/* This doesn't actually test the happy flow, since the mocked HTTP stuff
 * doesn't implement http.Hijacker. */

type HTTPHandlerTestSuite struct {
	fss *HTTPHandler
}

var _ = check.Suite(&HTTPHandlerTestSuite{})

func (s *HTTPHandlerTestSuite) SetUpTest(c *check.C) {
	httpmock.Activate()
	s.fss = CreateHTTPHandler(nil)
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
