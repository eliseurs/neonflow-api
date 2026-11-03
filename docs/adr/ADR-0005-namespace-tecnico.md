# ADR-0005 — Namespace técnico da vertical

- **Status:** proposta
- **Decisores:** fundador, engenharia
- **Relacionados:** PEN-37

## Contexto e problema

A R79 mantém separada a questão do namespace técnico `neoflow` (item 7 do Orquestrador). Um documento de segurança de maio/2026 exemplifica `neonflow:locacao`, mas não confirma que esteja implantado (Base v1.1 §12).

## Opções consideradas

1. Manter `neoflow` como namespace técnico e `Neon Flow` só como nome de apresentação.
2. Migrar para `neonflow`, com plano de migração de tabelas, permissões e eventos.
3. Adiar até o levantamento do Nexus/Replit (PEN-35, PEN-36).

## Decisão

**Não decidida.** Recomendação: opção 3, depois escolher entre 1 e 2 com base no que existir em produção. Até lá, nenhum código novo cria namespace.
