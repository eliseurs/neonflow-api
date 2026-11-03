# ADR-0003 — Protótipo estático é evidência, não arquitetura-alvo

- **Status:** aceita
- **Decisores:** fundador, engenharia
- **Relacionados:** PEN-22, PEN-40

## Contexto e problema

`dist/` (commit `fdf114ef`) é um site estático com dados em memória. O pacote de 09/10 orienta: "Não copie o protótipo como arquitetura definitiva" e "O código é evidência original" (00_LEIA_PRIMEIRO; Dossiê L.6).

## Opções consideradas

1. Evoluir `dist/` incrementalmente até virar o produto.
2. Manter `dist/` congelado como evidência e construir o produto sobre a stack e os serviços existentes do Nexus.

## Decisão

**Aceita em 03/11/2026.** Opção 2. `dist/` e `tests/legacy/` ficam congelados. O produto nasce da [plano de reserva persistente](../plano-reserva-persistente.md) sobre a stack definida em PEN-40. A demo pode continuar publicada como material de visão.

## Consequências

- Boas: preserva a evidência e evita carregar defeitos (PEN-25 a PEN-28) para produção.
- Ruins: as regras do protótipo precisam ser reimplementadas, guiadas pelas RNs.
