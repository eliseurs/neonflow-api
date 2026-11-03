> **Substituída por [ADR-0010](ADR-0010-documentacao-no-padrao-claude.md)** em 03/11/2026.

# ADR-0007 — Wiki docs-as-code no próprio repositório

- **Status:** substituida (ADR-0010)
- **Decisores:** engenharia
- **Relacionados:** PEN-31, PEN-47

## Contexto e problema

O conhecimento do Neon Flow estava no Drive, em capítulos do protótipo e em conversas. O template de wiki da Trinca (`project-wiki-template`) não estava disponível, e a instância encontrada pertence a outro cliente.

## Opções consideradas

1. Wiki em repositório separado, consumida como submodule.
2. Wiki em `wiki/` dentro do repositório da aplicação, importada pelo `CLAUDE.md`.

## Decisão

Opção 2, escolhida pelo time em 03/11/2026. A estrutura segue padrões de mercado (docs-as-code, MADR, C4/arc42, Gherkin) e as convenções da Trinca (IDs rastreáveis, status explícito, não inventar regra). A validação roda com `npm run wiki:validate`.

## Consequências

- Boas: fonte única, sem drift de clone. Documentação revisada no mesmo PR do código.
- Ruins: se outra vertical precisar desta wiki, será preciso extraí-la para um repositório próprio.
