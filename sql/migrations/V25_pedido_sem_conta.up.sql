-- Pedido feito na página do carro sem conta (RN-02.38): fica aguardando a confirmação do e-mail, invisível
-- para a central. Ao confirmar, a conta é ligada ao cadastro do cliente e o pedido vai para a central.
ALTER TABLE site_requests DROP CONSTRAINT IF EXISTS site_requests_status_check;
ALTER TABLE site_requests ADD CONSTRAINT site_requests_status_check
    CHECK (status IN ('aguardando_email', 'solicitado', 'ajuste_pedido', 'confirmado', 'recusado', 'cancelado'));

-- O cadastro do cliente só existe depois da confirmação (pode ser o da central, com o histórico).
ALTER TABLE site_requests ALTER COLUMN customer_id DROP NOT NULL;
ALTER TABLE site_requests ADD CONSTRAINT site_requests_customer_check
    CHECK (status = 'aguardando_email' OR customer_id IS NOT NULL);
