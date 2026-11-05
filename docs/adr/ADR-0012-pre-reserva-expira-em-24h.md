# ADR-0012 — Pré-reserva expira em 24 horas sem pagamento

- **Status:** aceita
- **Data:** 04/11/2026
- **Decisores:** fundador
- **Relacionados:** PEN-04, PEN-24, substitui [ADR-0006](ADR-0006-momento-do-bloqueio-de-reserva.md)

## Contexto e problema

O ADR-0006 decidiu bloquear o período na criação da reserva, sem expiração. Depois disso, o fundador definiu 24 horas para o cliente pagar (RN-02.8). As duas regras juntas deixavam em aberto o que acontece quando o prazo vence.

## Opções consideradas

1. Cancelar sozinha e liberar o veículo (opção 3 do ADR-0006: pré-reserva com validade).
2. Avisar a central, que decide.
3. Manter reservada até alguém cancelar (ADR-0006 como estava).

## Decisão

**Opção 1**, escolhida pelo fundador no questionário (q39). O período continua bloqueado desde a criação. Se o pagamento não for confirmado em 24 horas, a reserva é cancelada automaticamente e o período volta a ficar disponível.

## Consequências

- Boas: reservas abandonadas não prendem a frota.
- A API precisa de uma rotina que cancele as pré-reservas vencidas, e a consulta de disponibilidade deve ignorar as que já passaram do prazo, mesmo antes de a rotina rodar.
- O cancelamento automático não gera multa (não houve pagamento) e precisa ficar registrado no histórico da reserva, com o motivo.
- Falta definir se o cliente é avisado antes de vencer.

## Fontes

- Questionário "Decisões do Neon Flow", q17 (prazo de 24h, 03/11/2026) e q39 (cancela sozinha, 04/11/2026).
