package photo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cabeçalhos mínimos dos formatos aceitos.
var (
	JPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
	PNG  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	WEBP = []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
)

func TestDetectType(t *testing.T) {
	for _, tc := range []struct {
		head []byte
		ct   string
		ext  string
	}{{JPEG, "image/jpeg", "jpg"}, {PNG, "image/png", "png"}, {WEBP, "image/webp", "webp"}} {
		ct, ext, err := DetectType(tc.head)
		require.NoError(t, err)
		assert.Equal(t, tc.ct, ct)
		assert.Equal(t, tc.ext, ext)
	}
	_, _, err := DetectType([]byte("%PDF-1.7"))
	assert.ErrorIs(t, err, ErrInvalidType)
	_, _, err = DetectType([]byte("GIF89a"))
	assert.ErrorIs(t, err, ErrInvalidType, "GIF não é aceito")
}

func TestStageAllowed(t *testing.T) {
	assert.True(t, StageAllowed(StageDelivery, "confirmada"))
	assert.True(t, StageAllowed(StageDelivery, "em_locacao"))
	assert.False(t, StageAllowed(StageDelivery, "devolvida"))
	assert.True(t, StageAllowed(StageReturn, "em_locacao"))
	assert.True(t, StageAllowed(StageReturn, "devolvida"))
	assert.False(t, StageAllowed(StageReturn, "encerrada"))
	assert.False(t, StageAllowed(StageReturn, "cancelada"))
	assert.False(t, StageAllowed("outra", "devolvida"))
}
