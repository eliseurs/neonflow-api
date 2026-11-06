# Registro de pendências

**66 pendências, 15 bloqueantes** (33 resolvidas, 14 parciais, 19 abertas), levantadas em 03/11/2026 a partir do código e da documentação. Fonte única no repositório (`docs/pendencias.md`). A versão para leitura e comentários está no documento "Neon Flow — Pendências do projeto" (Claude Docs).

Severidade: **Bloqueante** impede operar com cliente real · **Alta** necessária para o primeiro ciclo persistente · **Média** precisa de decisão antes do MVP · **Baixa** higiene.

Para resolver uma pendência: registre a decisão (ADR ou RN), mude o status para `resolvida` com o link e a data, e anote a mudança em `.claude/specs.md`.

| Categoria | Total | Bloqueante | Alta | Média | Baixa |
| --- | --- | --- | --- | --- | --- |
| Negócio e regras | 18 | 6 | 10 | 2 | 0 |
| Produto e escopo | 16 | 1 | 6 | 9 | 0 |
| Técnicas | 14 | 4 | 6 | 2 | 2 |
| Arquitetura e Nexus | 7 | 2 | 3 | 2 | 0 |
| Documentais | 6 | 0 | 2 | 2 | 2 |
| Governança e qualidade | 5 | 2 | 2 | 1 | 0 |

## Negócio e regras operacionais

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-01 | Inventariar a frota real ofertável: veículos próprios e de parceiros, proprietário, documentos e situação | Bloqueante | Fundador / operação | parcial: Jetta 2014 (Economy) e Mercedes C180 (Premium), os dois de parceiros, 03–05/11/2026 (RN-01); de 2 a 5 motoristas no início (06/11/2026, q84, [RN-02.27](../.claude/business-rules/02-reservas.md)); caução de R$ 350 e 200 km por dia nos dois carros (q96, 06/11/2026); o valor do km excedente não foi informado (o limite já pode ser cadastrado sem ele); faltam documentos e situação | Base v1.1 §8.1; Dossiê L.4 |
| PEN-02 | Definir tarifas, período mínimo, limite de km, combustível, atraso e extensão | Bloqueante | Fundador | resolvida em 05/11/2026: valores configuráveis pela operação ([RN-01.5](../.claude/business-rules/01-frota.md)), mínimo de 1 diária, tanque cheio-cheio, com motorista sob consulta, km por dia configurável, 1h de tolerância e depois cobrança por minuto exato (diária ÷ 1440, q89 e CLI, 06/11/2026), extensão pela central ([RN-02.12 a 02.20](../.claude/business-rules/02-reservas.md)); em 06/11/2026: km livre ou limitado acordado na contratação, excedente cobrado (RN-02.18, q82; implementado com odômetro na entrega e na devolução, V7), diária em períodos de 24h (RN-02.24, q67); franquia de km só nas diárias contratadas, somada no período, e limite aceito sem o valor do excedente (q90 e CLI); devolução antecipada e cobrança dos extras decididas pela central (RN-02.28, RN-02.29, q91 e q92) | Base v1.1 §8.2 |
| PEN-03 | Definir requisitos do condutor, caução, cobertura/seguro, responsabilidade, multas e ocorrências | Bloqueante | Fundador + jurídico | parcial: 21 anos, CNH há 1 ano, estrangeiro com habilitação e passaporte, caução por pré-autorização no cartão com valor por veículo (cadastro implementado em 06/11/2026, V7), seguro incluso com franquia, multas repassadas ao condutor principal ([RN-06](../.claude/business-rules/06-condutor-caucao-e-seguro.md), 04–06/11/2026); falta o jurídico confirmar a habilitação estrangeira (PEN-50) | Base v1.1 §8.3 |
| PEN-04 | Decidir a política de reserva: momento do bloqueio, validade da pré-reserva e prazo de pagamento ([ADR-0006](adr/ADR-0006-momento-do-bloqueio-de-reserva.md)) | Bloqueante | Fundador | resolvida em 04/11/2026 ([ADR-0012](adr/ADR-0012-pre-reserva-expira-em-24h.md), RN-02.11): 24h para pagar, depois cancela sozinha e libera o veículo | Dossiê F e L.3 |
| PEN-05 | Definir a ordem entre contrato, assinatura, pagamento e caução e escolher o provedor de assinatura | Bloqueante | Fundador + engenharia | parcial: pagamento integral na reserva, por Pix ou link confirmado pela central no lançamento, aceite dos termos registrado, Stripe para o pagamento online e estorno manual pela central ([ADR-0013](adr/ADR-0013-pagamento-online-e-moedas.md), 04–06/11/2026). Conflito da q71 resolvido em 06/11/2026 pela q88: continua o pagamento integral na reserva, com cancelamento em 24h sem pagamento, e caução na retirada ([RN-02](../.claude/business-rules/02-reservas.md)) | Base v1.1 §5 e §8.5 |
| PEN-06 | Definir cancelamento, multa e estorno, inclusive de locação já paga | Bloqueante | Fundador + financeiro | resolvida em 05/11/2026: transfer segue a LuxuryBus (RN-02.10); aluguel grátis até 48h antes, depois 1 diária, restante estornado ([RN-02.21](../.claude/business-rules/02-reservas.md)). Prazo e meio do estorno dependem do provedor (PEN-05) | Dossiê F |
| PEN-07 | Definir regras de transfer: distribuição, aceite, substituição, atraso, cancelamento, escalada | Alta | Fundador / operação | parcial: 60 min de espera inclusos, hora extra cobrada, corrida oferecida e o primeiro motorista que aceita leva (RN-02.9, 02.16, 02.17, 04/11/2026); valor da hora configurável (RN-01.5); se ninguém aceitar, a oferta é reaberta com valor maior (RN-02.22, 05/11/2026); a corrida chega pelo WhatsApp da central e pelo painel ou app do motorista (RN-02.27, 06/11/2026); prazo de aceite configurável e acréscimo definido pela central, e desistência volta a oferecer sem penalidade (RN-02.30, RN-02.31, 06/11/2026, q93 a q95); falta implementar o módulo de transfer | Base v1.1 §8.6 |
| PEN-08 | Fixar comissões: percentual, base, elegibilidade, estorno. Conflito ~15% × 20%/15% | Alta | Fundador | parcial: percentual configurável por categoria (RN-03.4); paga depois do pagamento e do serviço finalizado, então cancelada não gera comissão (RN-03.8, 05/11/2026); faltam base de cálculo e elegibilidade (acordo com o hotel, PEN-65) | Revisão 29/09 §14 e §16 |
| PEN-09 | Definir remuneração de motoristas e proprietários/locadoras parceiras | Alta | Fundador | resolvida em 05/11/2026: o Neon Flow recebe, fica com percentual configurável e repassa ao dono (RN-03.7); motorista recebe percentual da corrida (RN-03.9); no transfer e no passeio o dono é o motorista (RN-03.10) ([RN-03](../.claude/business-rules/03-financeiro-e-comissoes.md)) | Dossiê D |
| PEN-10 | Decidir se a venda de página/bio (~R$400, 5–10 templates) entra no produto | Média | Fundador | aberta | Dossiê D |
| PEN-11 | Aprovar métricas e metas antes de expandir | Alta | Fundador | parcial: indicadores da tela inicial escolhidos em 06/11/2026 (q79): retiradas e devoluções do dia, reservas aguardando pagamento, faturamento do mês, ocupação da frota, repasse a parceiros, comissão de hotéis e avaliação dos clientes; fórmulas em 06/11/2026: faturamento do mês mostra lado a lado as reservas com retirada no mês (com atraso e km) e o que foi pago no mês (q97; exige registrar a data do pagamento), e ocupação = carros alugados hoje ÷ frota (q98); faltam metas, a avaliação (exige coletar avaliações) e repasse e comissão (PEN-08, PEN-09) | Revisão 29/09 §14 |
| PEN-12 | Validar mercado: concorrência, preço e demanda atuais | Média | Produto | aberta | Dossiê C |
| PEN-54 | Decidir se o aluguel é com motorista, sem motorista ou os dois | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): os dois | Linha visual LuxuryBus, 03/11/2026 |
| PEN-58 | Definir aeroportos e cidades atendidos (POA, CXJ, Gramado, Canela, outras da Serra) | Alta | Fundador / operação | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): lista base + demais cidades sob demanda | Linha visual LuxuryBus, 03/11/2026 |
| PEN-59 | Decidir se o preço fica público no site (tabela trecho × veículo) ou sob consulta | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): aluguel com preço visível, transfer sob consulta | Linha visual LuxuryBus, 03/11/2026 |
| PEN-61 | Definir as categorias de veículo e se a frota comporta posicionamento premium | Alta | Fundador | resolvida em 03/11/2026 ([RN-01.4](../.claude/business-rules/01-frota.md)): Economy e Premium | Linha visual LuxuryBus, 03/11/2026 |
| PEN-65 | Hotéis parceiros: acordos existentes, autorização de uso de nome e foto, papel (indicador ou ponto de embarque) | Alta | Fundador | parcial: app nasce com template predefinido, área de parceiros depois ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md), 03/11/2026); fotos ilustrativas por enquanto e cadastro do veículo com fotos (armazenamento proposto no [ADR-0014](adr/ADR-0014-armazenamento-de-fotos.md)) ([RN-01.7](../.claude/business-rules/01-frota.md), 06/11/2026, q85); faltam acordos e autorização de uso de nome e foto dos hotéis | Linha visual LuxuryBus, 03/11/2026 |
| PEN-66 | Informar o número de WhatsApp da central (canal de reserva do ADR-0011) | Alta | Fundador / operação | resolvida em 05/11/2026: +55 51 99719-2854, configurado em `VITE_WHATSAPP_NUMBER` do `neonflow-app` (Railway `hml`); a landing já abre o WhatsApp da central | Landing, 05/11/2026 |

## Produto e escopo

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-13 | Definir critérios para sair da demonstração e operar com clientes reais | Bloqueante | Fundador + produto | resolvida em 04/11/2026: indispensáveis no dia 1 são landing PT/EN, pedido por WhatsApp/formulário, cadastros de veículos, clientes e motoristas, reserva sem conflito, aceite das condições, checklist com fotos e comissão de hotéis. Pagamento online ficou de fora (ver ADR-0013) | Base v1.1 §8.10 |
| PEN-14 | Especificar a matriz de perfis e permissões | Alta | Produto | parcial: no início só o fundador usa o painel, como admin e central; parceiros (hotéis, proprietários, indicadores) e motoristas depois ([RN-05.6](../.claude/business-rules/05-autorizacao.md), 06/11/2026); gestão de usuários admin/central no painel implementada em 06/11/2026 (RN-05.7); falta a matriz dos perfis de parceiros | Base v1.1 §4 |
| PEN-15 | Especificar a área de clientes (contatos, documentos, preferências, histórico) | Alta | Produto | parcial: portal do cliente desde o início ([RN-04.6](../.claude/business-rules/04-dados-e-integracoes.md), 06/11/2026); depende de PEN-48 para guardar documentos | Dossiê I e J |
| PEN-16 | Especificar cadastro de motoristas e parceiros e edição de frota | Alta | Produto | parcial: cadastro de parceiros pela central com papéis múltiplos, dados básicos e percentuais por parceiro; veículo ligado ao parceiro proprietário ([RN-07](../.claude/business-rules/07-parceiros.md), 06/11/2026); faltam documentos (PEN-48), área do parceiro e motorista nas corridas | Dossiê I |
| PEN-17 | Especificar indicações: QR, atribuição de origem, resultado ao indicador | Alta | Produto | parcial: todo parceiro indica por link ou QR com código único; o código vai na mensagem do WhatsApp e a central registra "Indicado por" na reserva ([RN-07](../.claude/business-rules/07-parceiros.md), 06/11/2026); faltam o cálculo da comissão e o resultado ao indicador | Base v1.1 §5; Dossiê J |
| PEN-18 | Detalhar o checklist de entrega e devolução (fotos, combustível, km, avarias) | Média | Produto + operação | parcial: o veículo só volta a ficar disponível com o checklist de devolução aprovado; itens iniciais: combustível, estética, óleo, água, pastilhas, pneus, limpeza interna, documentos, fotos obrigatórias e km, configuráveis pela central e implementados em 06/11/2026 ([RN-01.6](../.claude/business-rules/01-frota.md), 05–06/11/2026); falta decidir o checklist de retirada (q80 cita a vistoria da Localiza na retirada) | Base v1.1 §5 |
| PEN-19 | Modelar proposta versionada, negociação e mudança de intenção | Média | Produto | aberta | Dossiê E e F |
| PEN-20 | Aprovar a sequência de entrada ([roadmap](produto/roadmap.md)) | Média | Fundador | aberta | Revisão 29/09 §14 |
| PEN-53 | Decidir o produto principal da home: aluguel, transfer ou os dois | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): aluguel e transfer com o mesmo peso | Linha visual LuxuryBus, 03/11/2026 |
| PEN-55 | Decidir se o shuttle compartilhado (horário fixo, tarifa por pessoa) entra | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): não entra | Linha visual LuxuryBus, 03/11/2026 |
| PEN-56 | Decidir se entram pacotes (transfer + hotel + passeio) | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): mais adiante (passeio + transfer) | Linha visual LuxuryBus, 03/11/2026 |
| PEN-57 | Decidir se haverá convênio corporativo com fatura mensal | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): não entra | Linha visual LuxuryBus, 03/11/2026 |
| PEN-60 | Definir como o cliente fecha a reserva: WhatsApp, formulário para a central ou pagamento online | Alta | Fundador + produto | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): WhatsApp, formulário e pagamento online | Linha visual LuxuryBus, 03/11/2026 |
| PEN-62 | Decidir entre fotos da frota real ou ilustrativas | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): mistas | Linha visual LuxuryBus, 03/11/2026 |
| PEN-63 | Definir idiomas e moedas do site | Média | Fundador | resolvida em 05/11/2026 ([ADR-0013](adr/ADR-0013-pagamento-online-e-moedas.md)): português e inglês; cobrança em reais, dólar só exibido como conversão | Linha visual LuxuryBus, 03/11/2026 |
| PEN-64 | Confirmar se existe logo do Neon Flow para fundo preto e dourado | Média | Fundador | resolvida em 05/11/2026: logo de `nepo/resources/logo.jpg` (anel, estrada e seta, com "neon flow" em Poppins), substituindo o caminho B; SVG redesenhado a partir da imagem, porque o fundador só tem a imagem (06/11/2026, q86). Arquivos em `neonflow-app/public/brand/` | Linha visual LuxuryBus, 03/11/2026 |

## Técnicas do código atual

| ID | Pendência | Severidade | Onde | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-21 | Sem persistência: arrays em memória, recarga apaga tudo | Bloqueante | `dist/app.js` | resolvida na API em 05/11/2026: frota e reservas no PostgreSQL (V2, V3); o protótipo continua em memória | Dossiê H; CT-08 |
| PEN-22 | Sem backend: validação só no navegador | Bloqueante | `dist/app.js` | resolvida na API em 05/11/2026: validação em `fleet` e `booking` | Dossiê I |
| PEN-23 | Sem autenticação nem controle de acesso por perfil | Bloqueante | — | resolvida na API em 05/11/2026: JWT e papéis admin/central em todas as rotas de frota e reserva; perfis do produto seguem em PEN-14 | Dossiê I |
| PEN-24 | Sem bloqueio atômico: duas sessões reservam o mesmo período | Bloqueante | `createBooking` (app.js:41) | resolvida na API em 05/11/2026: `EXCLUDE USING gist` em `bookings`, testado com 8 criações simultâneas | Dossiê F |
| PEN-25 | Data inexistente (30/02) aceita | Alta | `createBooking` (app.js:41) | resolvida na API em 05/11/2026 (RN-02.7) | CT-09 |
| PEN-26 | `advance` vai além da etapa 4 | Alta | `advance` (app.js:45) | resolvida na API em 05/11/2026 (RN-02.4) | CT-12 |
| PEN-27 | Total recalculado com a tarifa atual, sem snapshot | Alta | `total` (app.js:19) | resolvida na API em 05/11/2026 (RN-02.6) | CT-11 |
| PEN-28 | Veículo "Livre" com reserva futura confirmada | Alta | `vehicleState` (app.js:20) | resolvida em 06/11/2026: a frota mostra "Reservado" e a próxima reserva ([RN-01.3](../.claude/business-rules/01-frota.md), q81) e implementada no mesmo dia (`GET /api/vehicles` traz `status` e `next_booking`) | CT-10 |
| PEN-29 | Sem horário, intervalo de limpeza ou fuso; 09:00 fixo | Alta | `createBooking` | resolvida em 06/11/2026: horário livre combinado com a central, diária em períodos de 24h a partir da retirada, saída no mesmo dia com checklist aprovado, central 24h ([RN-02.23 a 02.26](../.claude/business-rules/02-reservas.md), q66–q69); fuso `America/Sao_Paulo` por suposição técnica; implementada no mesmo dia (`start_at`/`end_at`, V5); hora real da entrega e da devolução e atraso implementados em seguida (V6, RN-02.19) | Dossiê F |
| PEN-30 | Financeiro remove cancelada paga; sem caução, custos, conciliação, repasses | Alta | `finance` (app.js:29) | aberta | Dossiê F |
| PEN-31 | Testes legados fora do CI | Média | `prototipo/tests/legacy/` | resolvida em 06/11/2026: rodam no CI do `neonflow-app` (`.github/workflows/ci.yml`, passo "Testes do protótipo congelado") | 02_CENARIOS |
| PEN-32 | Sem lint, formatação nem CI | Média | raiz | resolvida em 06/11/2026: GitHub Actions em todo PR e push no `main`. API: gofmt, `go vet`, build, testes unitários e de integração com PostgreSQL 16 e cobertura mínima de 80% em `internal/`. App: TypeScript, ESLint, Vitest, build e testes do protótipo | Manifesto |
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
| PEN-48 | Tratamento LGPD de CNH, documentos e contatos ([seguranca-e-lgpd.md](seguranca-lgpd.md)) | Bloqueante | Fundador + jurídico | parcial: dados a guardar definidos e o fundador é o encarregado ([RN-04.4 e 04.5](../.claude/business-rules/04-dados-e-integracoes.md), 06/11/2026, q74 e q76); falta o prazo de retenção com o advogado (q75), a base legal e o controle de acesso. A API segue sem guardar documentos | Dossiê G e H |
| PEN-49 | Isolamento por tenant e por perfil | Bloqueante | Engenharia | aberta | Dossiê K |
| PEN-50 | Parecer jurídico, fiscal e regulatório (locação, transfer, intermediação) | Alta | Fundador + jurídico | resolvida em 04/11/2026 por declaração do fundador: "já consultei, e não precisa". Sem parecer anexado; habilitação estrangeira (RN-06.3): o fundador vai confirmar com o advogado (05/11/2026) | Dossiê C |
| PEN-51 | QA visual, teclado e acessibilidade | Média | QA | aberta | Dossiê I e K |
| PEN-52 | Autorização explícita para implementar (o pacote de 09/10 autoriza só documentação) | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): landing e reserva autorizadas | Dossiê L; 00_LEIA_PRIMEIRO |
