# ADR-0006 — Momento do bloqueio de disponibilidade

- **Status:** substituída por [ADR-0012](ADR-0012-pre-reserva-expira-em-24h.md) em 04/11/2026
- **Decisores:** time (03/11/2026)
- **Relacionados:** PEN-04, PEN-24

## Contexto e problema

O protótipo bloqueia o período ao **criar** a reserva (etapa 0), sem expiração. A Base v1.1 sugeria bloquear na **confirmação**. O Dossiê F pede que essa diferença seja decidida e não normalizada em silêncio.

## Opções consideradas

1. **Bloquear na criação, sem prazo** (protótipo atual). Simples, mas reservas abandonadas prendem a frota.
2. **Bloquear só na confirmação.** Libera a frota, mas permite que dois clientes avancem na documentação para o mesmo carro.
3. **Pré-reserva com prazo de validade (TTL)**, que vira bloqueio firme na confirmação. Padrão de mercado em locadoras e hotelaria. O prazo precisa ser definido pelo negócio.

## Decisão

**Aceita em 03/11/2026: opção 1, bloquear na criação, sem prazo.** Decisão do time ao iniciar o scaffold. Registrada como RN-02.1.

O bloqueio será atômico no banco (PEN-24), e não só no navegador como no protótipo.

## Consequências

- Boas: comportamento simples e igual ao protótipo. A central sabe que, criada a reserva, o carro está garantido.
- Ruins: reservas abandonadas prendem a frota até alguém cancelar. Se isso virar problema, um novo ADR pode introduzir a pré-reserva com validade (opção 3).
- O prazo comercial para pagamento continua em aberto (PEN-04 parcial).
