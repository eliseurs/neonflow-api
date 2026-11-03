# ADR-0001 — Registrar decisões em ADR no formato MADR

- **Status:** aceita
- **Decisores:** engenharia
- **Relacionados:** PEN-47

## Contexto e problema

As decisões do Neon Flow estão espalhadas em conversas, dossiês e capítulos do protótipo, misturando fala do fundador, proposta da IA e síntese (Dossiê B e M). Isso já causou erro de atribuição, como a regra de comissão (PEN-46).

## Decisão

Toda decisão de arquitetura ou de estrutura vira um ADR em `wiki/arquitetura/adr/`, no formato MADR, com status, decisores e fontes. Regras de negócio seguem o mesmo princípio em `RN-NNNN`.

## Consequências

- Boas: a autoria e o status de cada decisão ficam explícitos e versionados.
- Ruins: exige disciplina para registrar decisões tomadas fora do repositório.
