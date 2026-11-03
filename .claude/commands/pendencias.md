---
description: Lista ou filtra as pendências do Neon Flow (PEN-NN) por severidade, categoria ou responsável
argument-hint: "[bloqueante|alta|media|baixa|negocio|tecnica|PEN-NN]"
---

Leia `docs/pendencias.md`, a fonte única das pendências.

- Sem argumento: mostre o resumo por categoria e severidade e liste só as **Bloqueantes**.
- Com severidade ou categoria (`$ARGUMENTS`): filtre a tabela e mostre ID, pendência e quem resolve.
- Com um ID (`PEN-NN`): explique a pendência, cite a fonte e liste ADRs, RNs e SPECs da wiki que a referenciam (busque o ID em `docs/` e `.claude/business-rules/`).

Não invente status nem marque pendência como resolvida. Resolver uma pendência exige decisão registrada (ADR ou RN aprovada) e atualização da tabela pela pessoa responsável.
