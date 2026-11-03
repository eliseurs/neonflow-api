# Registro de pendências

**52 pendências, 15 bloqueantes**, levantadas em 03/11/2026 a partir do código e da documentação. Fonte única no repositório (`docs/pendencias.md`). A versão para leitura e comentários está no documento "Neon Flow — Pendências do projeto" (Claude Docs).

Severidade: **Bloqueante** impede operar com cliente real · **Alta** necessária para o primeiro ciclo persistente · **Média** precisa de decisão antes do MVP · **Baixa** higiene.

Para resolver uma pendência: registre a decisão (ADR ou RN), mude o status para `resolvida` com o link e a data, e anote a mudança em `.claude/specs.md`.

| Categoria | Total | Bloqueante | Alta | Média | Baixa |
| --- | --- | --- | --- | --- | --- |
| Negócio e regras | 12 | 6 | 4 | 2 | 0 |
| Produto e escopo | 8 | 1 | 4 | 3 | 0 |
| Técnicas | 14 | 4 | 6 | 2 | 2 |
| Arquitetura e Nexus | 7 | 2 | 3 | 2 | 0 |
| Documentais | 6 | 0 | 2 | 2 | 2 |
| Governança e qualidade | 5 | 2 | 2 | 1 | 0 |

## Negócio e regras operacionais

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-01 | Inventariar a frota real ofertável: veículos próprios e de parceiros, proprietário, documentos e situação | Bloqueante | Fundador / operação | aberta | Base v1.1 §8.1; Dossiê L.4 |
| PEN-02 | Definir tarifas, período mínimo, limite de km, combustível, atraso e extensão | Bloqueante | Fundador | aberta | Base v1.1 §8.2 |
| PEN-03 | Definir requisitos do condutor, caução, cobertura/seguro, responsabilidade, multas e ocorrências | Bloqueante | Fundador + jurídico | aberta | Base v1.1 §8.3 |
| PEN-04 | Decidir a política de reserva: momento do bloqueio, validade da pré-reserva e prazo de pagamento ([ADR-0006](adr/ADR-0006-momento-do-bloqueio-de-reserva.md)) | Bloqueante | Fundador | parcial: bloqueio na criação, sem expiração, decidido em 03/11/2026 (RN-02.1); falta o prazo de pagamento | Dossiê F e L.3 |
| PEN-05 | Definir a ordem entre contrato, assinatura, pagamento e caução e escolher o provedor de assinatura | Bloqueante | Fundador + engenharia | aberta | Base v1.1 §5 e §8.5 |
| PEN-06 | Definir cancelamento, multa e estorno, inclusive de locação já paga | Bloqueante | Fundador + financeiro | aberta | Dossiê F |
| PEN-07 | Definir regras de transfer: distribuição, aceite, substituição, atraso, cancelamento, escalada | Alta | Fundador / operação | aberta | Base v1.1 §8.6 |
| PEN-08 | Fixar comissões: percentual, base, elegibilidade, estorno. Conflito ~15% × 20%/15% | Alta | Fundador | aberta | Revisão 29/09 §14 e §16 |
| PEN-09 | Definir remuneração de motoristas e proprietários/locadoras parceiras | Alta | Fundador | aberta | Dossiê D |
| PEN-10 | Decidir se a venda de página/bio (~R$400, 5–10 templates) entra no produto | Média | Fundador | aberta | Dossiê D |
| PEN-11 | Aprovar métricas e metas antes de expandir | Alta | Fundador | aberta | Revisão 29/09 §14 |
| PEN-12 | Validar mercado: concorrência, preço e demanda atuais | Média | Produto | aberta | Dossiê C |

## Produto e escopo

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-13 | Definir critérios para sair da demonstração e operar com clientes reais | Bloqueante | Fundador + produto | aberta | Base v1.1 §8.10 |
| PEN-14 | Especificar a matriz de perfis e permissões | Alta | Produto | aberta | Base v1.1 §4 |
| PEN-15 | Especificar a área de clientes (contatos, documentos, preferências, histórico) | Alta | Produto | aberta | Dossiê I e J |
| PEN-16 | Especificar cadastro de motoristas e parceiros e edição de frota | Alta | Produto | aberta | Dossiê I |
| PEN-17 | Especificar indicações: QR, atribuição de origem, resultado ao indicador | Alta | Produto | aberta | Base v1.1 §5; Dossiê J |
| PEN-18 | Detalhar o checklist de entrega e devolução (fotos, combustível, km, avarias) | Média | Produto + operação | aberta | Base v1.1 §5 |
| PEN-19 | Modelar proposta versionada, negociação e mudança de intenção | Média | Produto | aberta | Dossiê E e F |
| PEN-20 | Aprovar a sequência de entrada ([roadmap](produto/roadmap.md)) | Média | Fundador | aberta | Revisão 29/09 §14 |

## Técnicas do código atual

| ID | Pendência | Severidade | Onde | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-21 | Sem persistência: arrays em memória, recarga apaga tudo | Bloqueante | `dist/app.js` | aberta | Dossiê H; CT-08 |
| PEN-22 | Sem backend: validação só no navegador | Bloqueante | `dist/app.js` | aberta | Dossiê I |
| PEN-23 | Sem autenticação nem controle de acesso por perfil | Bloqueante | — | aberta | Dossiê I |
| PEN-24 | Sem bloqueio atômico: duas sessões reservam o mesmo período | Bloqueante | `createBooking` (app.js:41) | aberta | Dossiê F |
| PEN-25 | Data inexistente (30/02) aceita | Alta | `createBooking` (app.js:41) | aberta | CT-09 |
| PEN-26 | `advance` vai além da etapa 4 | Alta | `advance` (app.js:45) | aberta | CT-12 |
| PEN-27 | Total recalculado com a tarifa atual, sem snapshot | Alta | `total` (app.js:19) | aberta | CT-11 |
| PEN-28 | Veículo "Livre" com reserva futura confirmada | Alta | `vehicleState` (app.js:20) | aberta | CT-10 |
| PEN-29 | Sem horário, intervalo de limpeza ou fuso; 09:00 fixo | Alta | `createBooking` | aberta | Dossiê F |
| PEN-30 | Financeiro remove cancelada paga; sem caução, custos, conciliação, repasses | Alta | `finance` (app.js:29) | aberta | Dossiê F |
| PEN-31 | Testes legados fora do CI | Média | `prototipo/tests/legacy/` | parcial: trazidos para o repo em 03/11/2026; falta CI | 02_CENARIOS |
| PEN-32 | Sem lint, formatação nem CI | Média | raiz | parcial: Makefile, ESLint (app) e `go vet` (api) em 03/11/2026; falta CI | Manifesto |
| PEN-33 | README cita 8 áreas (são 9); agenda com data fixa 27/09/2026 | Baixa | `README.md`, `central` (app.js:24) | aberta | Dossiê I |
| PEN-34 | Google Fonts externo; WebMCP nunca testado | Baixa | `style.css:1`, app.js:51 | aberta | Dossiê H |

## Arquitetura e integração com o Nexus OS

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-35 | Localizar os donos existentes de identidade/CTX, agenda, documentos, pagamentos, eventos e permissões ([ADR-0004](adr/ADR-0004-reutilizar-capacidades-nexus.md)) | Bloqueante | Engenharia | aberta | Dossiê L.2 |
| PEN-36 | Auditar o Replit | Bloqueante | Engenharia | aberta | Dossiê J; 00_LEIA_PRIMEIRO |
| PEN-37 | Resolver o namespace técnico `neoflow` × `neonflow:locacao` ([ADR-0005](adr/ADR-0005-namespace-tecnico.md)) | Alta | Fundador + arquitetura | aberta | Dossiê L.1 |
| PEN-38 | Especificar contratos de integração com Shell e Nexus ([integracoes.md](integracoes.md)) | Alta | Arquitetura | aberta | Dossiê G |
| PEN-39 | Formalizar fronteiras com NeoAuto, Neo360, NeoReferral e NeoPath | Média | Arquitetura | aberta | Dossiê G |
| PEN-40 | Definir a stack alvo a partir do que existe, sem reescrita imposta | Alta | Engenharia | resolvida em 03/11/2026 ([ADR-0009](adr/ADR-0009-stack-e-repositorios.md)): mesma stack do cognia/finder | Dossiê J |
| PEN-41 | Definir a IA contextual: fontes, ferramentas, provedor, avaliações, PALMO | Média | Arquitetura | aberta | Dossiê H |

## Documentais e de evidência

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-42 | Localizar `NEON_FLOW_cartao_executivo.pdf` | Alta | Fundador | aberta | Revisão 29/09 §16 |
| PEN-43 | Reconfirmar números históricos (35 motoristas, 30 ativos, R$1.500/dia, SLA 60s, 20%/15%, 4–5 visitas/dia) | Alta | Fundador | aberta | Revisão 29/09 §16 |
| PEN-44 | Reconciliar "Neon Flow" × "Neo Flow" no dossiê comum | Média | Fundador | aberta | 00_LEIA_PRIMEIRO |
| PEN-45 | Ler documentos citados e nunca abertos (Arquitetura_Nexus_OS_Completa.docx, Documento_Recuperacao_Projeto_NeoNoctual_NexusOS.docx, nexus_replit_snapshot.md) | Média | Engenharia | aberta | Dossiê M |
| PEN-46 | Registrar a retificação da regra de comissão no mapa funcional | Baixa | Produto | parcial: registrada em [RN-03.2](../.claude/business-rules/03-financeiro-e-comissoes.md); o protótipo segue igual | Dossiê C.5 |
| PEN-47 | Criar registros de decisão (ADRs) | Baixa | Engenharia | resolvida em 03/11/2026 ([ADR-0001](adr/ADR-0001-registrar-decisoes-em-adr.md)) | — |

## Governança, segurança, LGPD e qualidade

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-48 | Tratamento LGPD de CNH, documentos e contatos ([seguranca-e-lgpd.md](seguranca-lgpd.md)) | Bloqueante | Fundador + jurídico | aberta | Dossiê G e H |
| PEN-49 | Isolamento por tenant e por perfil | Bloqueante | Engenharia | aberta | Dossiê K |
| PEN-50 | Parecer jurídico, fiscal e regulatório (locação, transfer, intermediação) | Alta | Fundador + jurídico | aberta | Dossiê C |
| PEN-51 | QA visual, teclado e acessibilidade | Média | QA | aberta | Dossiê I e K |
| PEN-52 | Autorização explícita para implementar (o pacote de 09/10 autoriza só documentação) | Alta | Fundador | aberta | Dossiê L; 00_LEIA_PRIMEIRO |
