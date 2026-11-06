// Package timezone fixa o fuso da operação: Gramado, America/Sao_Paulo (suposição técnica, RN-02).
package timezone

import (
	"time"
	_ "time/tzdata" // a imagem Alpine não tem a base de fusos; ela vai embutida no binário
)

// Location é o fuso em que datas e horas sem fuso explícito são interpretadas e exibidas.
var Location = mustLoad("America/Sao_Paulo")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic("timezone: " + err.Error())
	}
	return loc
}

// Date devolve o dia (AAAA-MM-DD) de t no fuso da operação.
func Date(t time.Time) string {
	return t.In(Location).Format("2006-01-02")
}
