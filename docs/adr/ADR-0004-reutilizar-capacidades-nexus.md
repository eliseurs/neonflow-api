# ADR-0004 — Reutilizar capacidades do Nexus antes de criar serviços

- **Status:** proposta
- **Decisores:** fundador, engenharia
- **Relacionados:** PEN-35, PEN-36

## Contexto e problema

O Nexus OS já declara capacidades compartilhadas de identidade, contexto, eventos, governança e memória. As fontes alertam contra criar uma "Engine" nova por diferença de nome e contra criar cinco serviços a partir das cinco primitivas (Dossiê C.6 e L; 00_LEIA_PRIMEIRO).

## Decisão (recomendada)

Para cada capacidade (identidade/CTX, agenda/disponibilidade, documentos, pagamentos, eventos, permissões), localizar primeiro o dono existente e registrar arquivo, função, commit e teste. Depois decidir REUSE, ADAPT ou EXTEND. Só criar algo novo com um ADR que justifique.

O Neon Flow mantém no próprio domínio: disponibilidade, reservas, regras de locação e operação de veículos (Base v1.1 §6).

## Consequências

- Boas: evita duplicar o Core e criar orquestradores concorrentes.
- Ruins: bloqueia a implementação até o levantamento (PEN-35, PEN-36).
