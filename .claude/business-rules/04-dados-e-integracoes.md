# RN-04 — Dados e Integrações

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-04.1 | Ao encaminhar uma intenção para outra vertical (Neo360, NeoAuto, NeoPath), só vão os dados necessários à finalidade e com autorização do cliente. CNH, caução e a conversa inteira nunca vão | proposta | — | PEN-39, PEN-48 |
| RN-04.2 | Identidade, contexto (CTX) e eventos são do Nexus Core. Antes de criar serviço próprio, localizar o dono existente | proposta ([ADR-0004](../../docs/adr/ADR-0004-reutilizar-capacidades-nexus.md)) | — | PEN-35, PEN-36 |
| RN-04.3 | Origem rastreada não equivale a comissão (ver RN-03.2) | proposta | — | PEN-08 |
| RN-04.4 | Dados do cliente guardados no aluguel: **nome, WhatsApp, e-mail, CPF ou passaporte, número e foto da CNH, data de nascimento e endereço** | aprovada (06/11/2026, fundador, q74); **retenção pendente** com o advogado (q75) | contato guardado desde a fase 1 (RN-04.7); documentos ainda não | PEN-48 |
| RN-04.5 | O **encarregado de dados (LGPD)** é o fundador | aprovada (06/11/2026, fundador, q76) | — | PEN-48 |
| RN-04.6 | O cliente tem **conta no site (portal do cliente) desde o início** | aprovada (06/11/2026, fundador, q77) | — | PEN-15, PEN-48 |
| RN-04.7 | **Cadastro de clientes, fase 1:** nome, WhatsApp (um cadastro por número), e-mail, idioma (PT ou EN) e observações, com o histórico de reservas e transfers. Reserva e transfer apontam para o cliente e guardam o nome do dia do pedido. Documentos, data de nascimento e endereço (RN-04.4) **ficam de fora** até a retenção ser definida | implementada (06/11/2026; o fundador pediu "Comece pelo cadastro de clientes"; o corte da fase 1 é decisão técnica para não guardar documentos sem PEN-48) | `V15_customers.up.sql`, `internal/domain/customer`, `bookings.customer_id`, `transfers.customer_id` | PEN-15, PEN-48 |
| RN-04.8 | **Anonimização a pedido do titular** (LGPD, art. 18): o admin, que é o encarregado (RN-04.5), apaga nome e contatos do cadastro e o nome gravado nos pedidos. Datas e valores ficam, como registro financeiro. Não se desfaz | implementada (06/11/2026; proposta técnica: a base legal para manter os registros financeiros precisa ser confirmada com o advogado) | `POST /api/customers/:id/anonymize` (só admin), `customer_repository.go` (`queryAnonymize`) | PEN-48 |

**Fontes:** Dossiê G; Revisão 29/09 §15; questionário do fundador em 06/11/2026 (RN-04.4 a RN-04.6); CLI em 06/11/2026 (RN-04.7, RN-04.8).

## Detalhes

- Interesse imobiliário durante a viagem: só com aprovação do cliente, e não se cria lead por inferência de turismo.
- Veículo alugado não vira oferta de venda no NeoAuto. Dados do locatário não são replicados.
- Contratos de integração propostos: `docs/integracoes.md`. Nenhuma integração existe hoje.
- RN-04.4: na q74 o fundador colou a política da Localiza, que também coleta categoria e validade da CNH, condutores adicionais, dados do cartão para pré-autorização, consulta de crédito (SCR), telemetria e dados de navegação. **Só os itens marcados** entram na regra; o resto é referência, não decisão.
- RN-04.4 × RN-04.6: foto da CNH e documentos só podem ser guardados com base legal, finalidade, prazo de retenção e controle de acesso definidos (PEN-48). Até lá, a API continua sem armazenar documentos. O portal do cliente depende disso (PEN-15).
