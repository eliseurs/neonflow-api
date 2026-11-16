-- Pedidos que nunca tiveram o e-mail confirmado não existem antes desta versão.
DELETE FROM site_request_events WHERE request_id IN (SELECT id FROM site_requests WHERE status = 'aguardando_email');
DELETE FROM site_requests WHERE status = 'aguardando_email';
ALTER TABLE site_requests DROP CONSTRAINT IF EXISTS site_requests_customer_check;
ALTER TABLE site_requests ALTER COLUMN customer_id SET NOT NULL;
ALTER TABLE site_requests DROP CONSTRAINT IF EXISTS site_requests_status_check;
ALTER TABLE site_requests ADD CONSTRAINT site_requests_status_check
    CHECK (status IN ('solicitado', 'ajuste_pedido', 'confirmado', 'recusado', 'cancelado'));
