package photo

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registra o decodificador PNG
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registra o decodificador WebP
)

const (
	// VariantQuality: JPEG 85 com reamostragem CatmullRom guarda os detalhes sem pesar.
	VariantQuality = 85
	// MaxPixels limita a foto decodificada (40 megapixels), para não estourar a memória.
	MaxPixels = 40_000_000
)

// VariantWidths são as larguras oferecidas ao navegador no srcset da landing.
var VariantWidths = []int{640, 1280, 1920}

// cardBackground é o fundo do cartão da landing, usado onde a foto (PNG/WebP) é transparente.
var cardBackground = color.RGBA{R: 0x0b, G: 0x0b, B: 0x0b, A: 0xff}

// Car é um carro da frota para o site público, com a foto de capa.
type Car struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Seats    int     `json:"seats"`
	Gear     string  `json:"gear"`
	PhotoID  *string `json:"photo_id"`
}

func VariantAllowed(width int) bool {
	for _, w := range VariantWidths {
		if w == width {
			return true
		}
	}
	return false
}

// VariantKey é onde a versão fica no bucket, ao lado da original: "veiculos/v1/abc-w640.jpg".
func VariantKey(key string, width int) string {
	if i := strings.LastIndex(key, "."); i > strings.LastIndex(key, "/") {
		key = key[:i]
	}
	return fmt.Sprintf("%s-w%d.jpg", key, width)
}

// Resize gera a versão em JPEG com até width pixels de largura, sem ampliar a original.
func Resize(data []byte, width int) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidType
	}
	if cfg.Width*cfg.Height > MaxPixels || cfg.Width == 0 || cfg.Height == 0 {
		return nil, ErrTooLarge
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidType
	}
	b := src.Bounds()
	w := min(width, b.Dx())
	h := max(1, b.Dy()*w/b.Dx())
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: cardBackground}, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: VariantQuality}); err != nil {
		return nil, fmt.Errorf("encode variant: %w", err)
	}
	return out.Bytes(), nil
}
