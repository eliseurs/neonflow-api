package checklist

import (
	"strings"
	"time"
	"unicode"

	"neonflow-api/internal/domain/booking"
)

// Item é um item do checklist de devolução (RN-01.6), configurável pela central.
// Key vira o nome da conferência na reserva em "devolvida" e não muda depois de criada.
type Item struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	Position  int       `json:"position"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ItemRequest é o corpo de criação e de atualização. Active ausente na criação vale true.
type ItemRequest struct {
	Label    string `json:"label"`
	Position int    `json:"position"`
	Active   *bool  `json:"active"`
}

const maxKeyLength = 40

var accents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// KeyFromLabel gera a chave a partir do nome: minúsculas, sem acento, só letras, números e "_".
func KeyFromLabel(label string) string {
	s := accents.Replace(strings.ToLower(strings.TrimSpace(label)))
	var b strings.Builder
	underscore := false
	for _, r := range s {
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
			underscore = false
			continue
		}
		if b.Len() > 0 && !underscore {
			b.WriteByte('_')
			underscore = true
		}
	}
	key := strings.Trim(b.String(), "_")
	if len(key) > maxKeyLength {
		key = strings.TrimRight(key[:maxKeyLength], "_")
	}
	return key
}

// Validate normaliza o nome e exige que ele não fique vazio.
func (r ItemRequest) Validate() (ItemRequest, error) {
	r.Label = strings.TrimSpace(r.Label)
	if r.Label == "" {
		return r, ErrLabelRequired
	}
	return r, nil
}

// NewKey gera e confere a chave de um item novo: não pode ficar vazia nem repetir uma conferência fixa da reserva.
func NewKey(label string) (string, error) {
	key := KeyFromLabel(label)
	if key == "" {
		return "", ErrInvalidLabel
	}
	for _, reserved := range booking.ReservedCheckKeys() {
		if key == reserved {
			return "", ErrKeyReserved
		}
	}
	return key, nil
}
