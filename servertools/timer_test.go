package servertools

import (
	"time"

	"github.com/stretchr/testify/assert"

	check "gopkg.in/check.v1"
)

type TimerTestSuite struct {
}

var _ = check.Suite(&TimerTestSuite{})

func readTimeout(t *check.C, timeout chan bool) (timeoutOccurred bool) {
	now := time.Now()
	timeoutOccurred = <-timeout
	passed := time.Now().Sub(now)

	if passed > 100*time.Millisecond {
		assert.Fail(t, "reading timeout status took too long", "took %s", passed)
	}

	return
}

func (s *TimerTestSuite) TestTimerNormal(t *check.C) {
	timeout := TimeoutAfter(1 * time.Second)
	timeout <- false

	assert.False(t, readTimeout(t, timeout))
}

func (s *TimerTestSuite) TestTimerChannelClosed(t *check.C) {
	timeout := TimeoutAfter(1 * time.Second)
	close(timeout)

	assert.False(t, readTimeout(t, timeout))
}

func (s *TimerTestSuite) TestTimerTimeout(t *check.C) {
	timeout := TimeoutAfter(10 * time.Microsecond)
	time.Sleep(100 * time.Microsecond)

	assert.True(t, readTimeout(t, timeout))
}
