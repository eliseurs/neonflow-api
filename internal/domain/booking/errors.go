package booking

import "errors"

var (
	ErrBookingNotFound      = errors.New("reserva não encontrada")
	ErrVehicleNotFound      = errors.New("veículo não encontrado")
	ErrVehicleInMaintenance = errors.New("veículo em manutenção não aceita reservas")
	ErrPeriodUnavailable    = errors.New("período indisponível para este veículo")
	ErrVehicleRequired      = errors.New("veículo é obrigatório")
	ErrClientNameRequired   = errors.New("nome do cliente é obrigatório")
	ErrInvalidDate          = errors.New("data e hora inválidas: use AAAA-MM-DDTHH:MM")
	ErrInvalidPeriod        = errors.New("a devolução deve ser depois da retirada")
	ErrInvalidCheck         = errors.New("conferência inválida")
	ErrChecksPending        = errors.New("conferências pendentes")
	ErrStageFinal           = errors.New("reserva encerrada ou cancelada não muda de etapa")
	ErrNotCancelable        = errors.New("só reservas em preparação ou confirmadas podem ser canceladas")
	ErrStageChanged         = errors.New("a reserva mudou de etapa; recarregue e tente de novo")
	ErrAwaitingChecklist    = errors.New("o veículo aguarda o checklist de devolução de outra reserva")
	ErrVehicleInUse         = errors.New("o veículo ainda está em locação em outra reserva")
	ErrFutureTime           = errors.New("a hora informada ainda não chegou")
	ErrReturnBeforeDelivery = errors.New("a devolução precisa ser depois da entrega")
)
