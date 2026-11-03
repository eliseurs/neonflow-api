package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	newWithWriter(true, &buf).Info("oi", "k", "v")
	assert.Contains(t, buf.String(), `"msg":"oi"`)

	buf.Reset()
	newWithWriter(false, &buf).Debug("dev")
	assert.Contains(t, buf.String(), "msg=dev")

	assert.NotNil(t, New(false))
}
