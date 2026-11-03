---
description: Cria um ADR (formato MADR) em docs/adr a partir do template
argument-hint: "<título da decisão>"
---

Crie um novo ADR para: $ARGUMENTS

1. Descubra o próximo número livre em `docs/adr/` (formato `ADR-NNNN`).
2. Copie `docs/adr/template.md` para `docs/adr/ADR-NNNN-<slug-kebab>.md`.
3. Preencha contexto, opções e consequências **só com fatos do código, dos docs ou do que o usuário disser**. Cite a fonte de cada fato.
4. Status inicial: `proposta`. Só o responsável marca `aceita`.
5. Liste as pendências relacionadas (`PEN-NN`) em "Relacionados".
6. Adicione a linha no índice `docs/adr/README.md` e registre em `.claude/specs.md`.

Para decisões estruturais relevantes, considere a skill `trinca-toolbox:adr-writer` (entrevista guiada).
