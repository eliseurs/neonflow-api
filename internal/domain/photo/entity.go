// Package photo são as fotos do veículo (RN-01.7) e da reserva na entrega e na devolução (RN-01.6).
// Os arquivos ficam no bucket privado (ADR-0014) e são lidos por link temporário.
package photo

import (
	"net/http"
	"time"
)

const (
	OwnerVehicle = "veiculo"
	OwnerBooking = "reserva"

	StageDelivery = "entrega"
	StageReturn   = "devolucao"

	// MaxSize é o limite de cada foto: 10 MB.
	MaxSize = 10 << 20
	// LinkTTL é a validade do link de leitura.
	LinkTTL = 15 * time.Minute
)

// types: formatos aceitos, detectados pelo conteúdo (não pela extensão do arquivo).
var types = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}

type Photo struct {
	ID          string    `json:"id"`
	OwnerType   string    `json:"owner_type"`
	OwnerID     string    `json:"owner_id"`
	Stage       string    `json:"stage"`
	ContentType string    `json:"content_type"`
	SizeBytes   int       `json:"size_bytes"`
	Position    int       `json:"position"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	// Key é a chave no bucket; não sai na API.
	Key string `json:"-"`
}

// NewPhoto é o que o repository grava.
type NewPhoto struct {
	OwnerType   string
	OwnerID     string
	Stage       string
	Key         string
	ContentType string
	SizeBytes   int
	CreatedBy   string
}

// DetectType confere o formato pelos primeiros bytes e devolve o tipo e a extensão.
func DetectType(head []byte) (contentType, ext string, err error) {
	ct := http.DetectContentType(head)
	ext, ok := types[ct]
	if !ok {
		return "", "", ErrInvalidType
	}
	return ct, ext, nil
}

// StageAllowed: a foto da entrega vale até a locação começar; a da devolução, na devolução
// e enquanto o checklist não é aprovado. Reserva encerrada ou cancelada não recebe foto.
func StageAllowed(photoStage, bookingStage string) bool {
	switch photoStage {
	case StageDelivery:
		return bookingStage == "confirmada" || bookingStage == "em_locacao"
	case StageReturn:
		return bookingStage == "em_locacao" || bookingStage == "devolvida"
	}
	return false
}
