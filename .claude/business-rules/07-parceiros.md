# RN-07 — Parceiros e indicações

Parceiros são os **proprietários de veículo**, os **motoristas** e os **indicadores** (hotéis, comércio). Na fase 1 são cadastrados e operados pela central; a área do próprio parceiro, com login, vem depois (RN-05.6, [ADR-0011](../../docs/adr/ADR-0011-escopo-de-lancamento-e-canais.md)).

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-07.1 | Um parceiro pode ter **mais de um papel** (proprietário, motorista, indicador) e **todo parceiro pode indicar** o serviço | implementada (06/11/2026; fundador: "Todos podem indicar ou seja todos podem ser parceiros e indicar o serviço seja através de link ou qrcode") | `V9_partners.up.sql` (`is_owner`, `is_driver`, `is_referrer`, ao menos um), `internal/domain/partner` | PEN-16 |
| RN-07.2 | A indicação é feita por **link ou QR code** com o código do parceiro. Quem chega pelo link tem o código guardado durante a visita, e ele vai na mensagem do WhatsApp ("Indicação: CÓDIGO") | implementada (06/11/2026, mesma resposta) | `neonflow-app`: `captureReferral` e `ReferralCard` (QR gerado no navegador, sem serviço externo) | PEN-17 |
| RN-07.3 | Cada parceiro recebe um **código de indicação único**, gerado na criação e que não muda (6 caracteres, sem 0/O/1/I/L) | implementada (06/11/2026; decisão técnica) | `partner.NewReferralCode`, índice único `partners_referral_code_key` | PEN-17 |
| RN-07.4 | O veículo pode apontar para o **parceiro proprietário** cadastrado; o nome livre do proprietário continua | implementada (06/11/2026) | `vehicles.owner_partner_id` (V9) | PEN-16 |
| RN-07.5 | A reserva registra **quem indicou** (origem). Isso **não gera comissão sozinho** (RN-03.2); o cálculo vem na fase 2 | implementada (06/11/2026) | `bookings.referred_by_partner_id` (V9); a central escolhe em "Indicado por" | PEN-17, PEN-08 |
| RN-07.6 | Os **percentuais são por parceiro** e configuráveis: comissão de indicação (RN-03.4), repasse ao proprietário (RN-03.7) e percentual do motorista (RN-03.9). Sem padrão no código; vazio = a definir | implementada (06/11/2026; fundador: "Por parceiro") | `commission_bp`, `owner_share_bp`, `driver_share_bp` em pontos-base | PEN-08, PEN-09 |
| RN-07.7 | O cadastro guarda **só dados básicos** (nome, WhatsApp, e-mail, cidade, observações). Documentos e chave Pix esperam a política da LGPD | implementada (06/11/2026; fundador: "Básico, sem documentos") | `partner.Request.Validate` | PEN-48 |

**Fontes:** respostas do fundador em 06/11/2026 no CLI (tipos, dados e percentual); `docs/produto/perfis.md`; RN-03.4 a RN-03.10; RN-05.6.

## Pontos em aberto (fase 2)

- Cálculo da comissão e do repasse por reserva, relatório e marcação de pago (RN-03.8: só depois do pagamento e do serviço finalizado).
- Área do parceiro com login e resultado das indicações (PEN-14, PEN-17).
- Ligação do motorista às corridas de transfer.
- Atribuição automática: hoje a central escolhe o indicador lendo o código da mensagem; com reserva online, o código iria direto para a reserva.
- Acordos com hotéis e uso de nome e foto (PEN-65).
