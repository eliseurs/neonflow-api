# RN-04 — Dados e Integrações

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-04.1 | Ao encaminhar uma intenção para outra vertical (Neo360, NeoAuto, NeoPath), só vão os dados necessários à finalidade e com autorização do cliente. CNH, caução e a conversa inteira nunca vão | proposta | — | PEN-39, PEN-48 |
| RN-04.2 | Identidade, contexto (CTX) e eventos são do Nexus Core. Antes de criar serviço próprio, localizar o dono existente | proposta ([ADR-0004](../../docs/adr/ADR-0004-reutilizar-capacidades-nexus.md)) | — | PEN-35, PEN-36 |
| RN-04.3 | Origem rastreada não equivale a comissão (ver RN-03.2) | proposta | — | PEN-08 |

**Fontes:** Dossiê G; Revisão 29/09 §15.

## Detalhes

- Interesse imobiliário durante a viagem: só com aprovação do cliente, e não se cria lead por inferência de turismo.
- Veículo alugado não vira oferta de venda no NeoAuto. Dados do locatário não são replicados.
- Contratos de integração propostos: `docs/integracoes.md`. Nenhuma integração existe hoje.
