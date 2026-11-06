package photo

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tinyWebP é um WebP sem perdas de 1x1 pixel.
const tinyWebP = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func sample(t *testing.T, w, h int, encode func(*bytes.Buffer, image.Image) error) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, encode(&buf, img))
	return buf.Bytes()
}

func size(t *testing.T, data []byte) (int, int, string) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	return cfg.Width, cfg.Height, format
}

func TestResize(t *testing.T) {
	jpg := sample(t, 2400, 1600, func(b *bytes.Buffer, i image.Image) error { return jpeg.Encode(b, i, &jpeg.Options{Quality: 95}) })
	out, err := Resize(jpg, 640)
	require.NoError(t, err)
	w, h, format := size(t, out)
	assert.Equal(t, 640, w)
	assert.Equal(t, 426, h, "mantém a proporção 3:2")
	assert.Equal(t, "jpeg", format)
	assert.Less(t, len(out), len(jpg))

	// Não amplia: a original de 800 px sai com 800 px na versão de 1920.
	small := sample(t, 800, 600, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) })
	out, err = Resize(small, 1920)
	require.NoError(t, err)
	w, h, format = size(t, out)
	assert.Equal(t, 800, w)
	assert.Equal(t, 600, h)
	assert.Equal(t, "jpeg", format, "PNG vira JPEG")

	webp, err := base64.StdEncoding.DecodeString(tinyWebP)
	require.NoError(t, err)
	out, err = Resize(webp, 640)
	require.NoError(t, err)
	w, _, format = size(t, out)
	assert.Equal(t, 1, w)
	assert.Equal(t, "jpeg", format, "WebP é lido e sai em JPEG")

	_, err = Resize([]byte("não é imagem"), 640)
	assert.ErrorIs(t, err, ErrInvalidType)
	// Cabeçalho de PNG gigante (50 mil x 50 mil): recusado antes de decodificar.
	huge := sample(t, 1, 1, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) })
	huge[16], huge[17], huge[18], huge[19] = 0, 0, 0xC3, 0x50
	huge[20], huge[21], huge[22], huge[23] = 0, 0, 0xC3, 0x50
	binary.BigEndian.PutUint32(huge[29:33], crc32.ChecksumIEEE(huge[12:29])) // CRC do IHDR
	_, err = Resize(huge, 640)
	assert.ErrorIs(t, err, ErrTooLarge)
	// Cabeçalho válido com dados corrompidos.
	broken := append([]byte{}, small[:40]...)
	_, err = Resize(broken, 640)
	assert.ErrorIs(t, err, ErrInvalidType)
}

func TestVariantKeyAndWidths(t *testing.T) {
	assert.Equal(t, "veiculos/v1/abc-w640.jpg", VariantKey("veiculos/v1/abc.webp", 640))
	assert.Equal(t, "veiculos/v.1/abc-w1280.jpg", VariantKey("veiculos/v.1/abc", 1280))
	assert.True(t, VariantAllowed(1920))
	assert.False(t, VariantAllowed(1000))
}
