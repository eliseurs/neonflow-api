-- Data do pagamento da reserva: gravada quando a central marca a conferência "payment" e apagada se
-- ela desmarcar. Alimenta o "pago no mês" da tela inicial (PEN-11, q97).
ALTER TABLE bookings ADD COLUMN paid_at TIMESTAMPTZ;

-- Reservas já pagas: a melhor aproximação é a confirmação (que exige o pagamento, RN-02.3);
-- sem ela, a última alteração da reserva.
UPDATE bookings b
SET paid_at = coalesce(
    (SELECT min(e.created_at) FROM booking_events e WHERE e.booking_id = b.id AND e.to_stage = 'confirmada'),
    b.updated_at)
WHERE (b.checks->>'payment')::boolean IS TRUE;
