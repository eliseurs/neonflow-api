-- Despacho do transfer como operação pequena por WhatsApp (06/11/2026): a central chama os motoristas
-- um a um e marca quem aceitou, sem prazo de oferta (RN-02.30 revista). O carro da corrida é registrado
-- à parte, porque o motorista nem sempre é o dono do carro (RN-03.10 revista).
ALTER TABLE transfers
    DROP CONSTRAINT transfers_offer_check,
    ADD COLUMN vehicle_id UUID REFERENCES vehicles (id);

UPDATE transfers SET offer_expires_at = NULL;

-- Histórico gravado em centavos ("35000 centavos até 06/11 15:42") passa a reais.
UPDATE transfer_events
SET detail = 'R$ ' || replace(to_char(substring(detail from '^(\d+) centavos')::numeric / 100, 'FM9999999990.00'), '.', ',')
WHERE detail ~ '^\d+ centavos';
