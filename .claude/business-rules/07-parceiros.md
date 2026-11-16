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
| RN-07.8 | **Área do parceiro, só leitura:** o parceiro logado (RN-05.12) vê o próprio código, link e QR de indicação e as seções dos tipos que tem. Vê sempre só o que é dele; os valores mostram o que ele recebe, com a base, o percentual e se já foi pago | implementada na API (16/11/2026, fundador no CLI: "base e percentual também"; resolve a parte de PEN-14 sobre o que o parceiro vê) | `internal/domain/partnerarea`, `GET /api/partner/area` | PEN-14 |
| RN-07.9 | **Proprietário:** vê os carros dele e as reservas desses carros (período, situação, diárias, atraso, km excedente e o repasse). **Sem nome nem contato do cliente** | implementada na API (16/11/2026, fundador no CLI: "datas e valores, sem o cliente") | `partnerarea_repository.go` (`queryBookings`) | PEN-48 |
| RN-07.10 | **Motorista:** vê as corridas dele com o necessário para executar (horário, origem, destino, voo, passageiros, nome do cliente) e o que recebe, e a **oferta aberta** com as corridas aguardando motorista para aceitar (RN-02.30). Pode desistir da que aceitou antes de começar (RN-02.31). O telefone e as observações do cliente ficam com a central | implementada na API (16/11/2026, fundador no CLI: "tudo para executar") | `partnerarea_repository.go` (`queryRides`) | PEN-74 |
| RN-07.11 | **Indicações:** todo parceiro vê as reservas e os transfers que indicou (serviço, data, situação e comissão), **sem o cliente** | implementada na API (16/11/2026, fundador no CLI: "serviço, data e comissão"; resolve a parte de PEN-17 sobre mostrar o resultado) | `partnerarea_repository.go` (`queryReferrals`) | PEN-17 |

**Fontes:** CLI em 16/11/2026 (RN-07.8 a RN-07.11, área do parceiro); respostas do fundador em 06/11/2026 no CLI (tipos, dados e percentual); `docs/produto/perfis.md`; RN-03.4 a RN-03.10; RN-05.6.

## Pontos em aberto (fase 2)

- ~~Cálculo da comissão e do repasse por reserva, relatório e marcação de pago~~: feito em 06/11/2026 (RN-03.12, RN-03.13, tela Repasses).

- ~~Área do parceiro com login e resultado das indicações~~: feito em 16/11/2026 (RN-05.12, RN-07.8 a RN-07.11).
- ~~Ligação do motorista às corridas de transfer~~: feita (V13, V14); aceite pelo próprio motorista segue na PEN-74.
- ~~Atribuição automática~~: o pedido do site leva o código direto (RN-02.33, 16/11/2026); pelo WhatsApp a central continua lendo o código.
- Acordos com hotéis e uso de nome e foto (PEN-65).
