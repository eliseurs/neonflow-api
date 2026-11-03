# ADR-0010 — Documentação no padrão `.claude/` + `docs/`

- **Status:** aceita
- **Data:** 03/11/2026
- **Decisores:** time
- **Substitui:** [ADR-0007](ADR-0007-wiki-docs-as-code-no-repositorio.md)

## Contexto e problema

A wiki em `wiki/` (ADR-0007) usava convenções próprias (frontmatter, IDs `RN-NNNN`, validador). Os outros projetos do time (cognia-api, cognia-app, clinic-app) usam outro formato, e o time pediu para padronizar.

## Decisão

| Conteúdo | Onde | Formato |
|---|---|---|
| Diretrizes de comportamento para agentes | `CLAUDE.md` (raiz) e `AGENTS.md` | Como o cognia |
| Índice, comandos e arquitetura | `.claude/CLAUDE.md` | Tabela de arquivos de referência |
| Regras técnicas | `.claude/rules.md` | Sempre ler |
| Arquitetura e stack | `.claude/design.md` | |
| Estilo de código | `.claude/style.md` | |
| Changelog e planos em andamento | `.claude/specs.md` | Atualizar após cada mudança |
| Regras de negócio | `.claude/business-rules/NN-tema.md` | Tabelas `RN-NN.x` com "onde é aplicada" e C4 |
| ADRs, pendências, produto, deploy, qualidade, fontes | `docs/` | Markdown simples |

Os IDs `RN-0001…RN-0009` foram renumerados para `RN-01.1…RN-04.1`. O validador da wiki foi removido.

## Consequências

- Boas: mesmo formato em todos os projetos.
- Ruins: perde-se a checagem automática de links e IDs que o validador fazia.
