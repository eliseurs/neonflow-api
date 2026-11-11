# Registro de pendências

**77 pendências, 16 bloqueantes** (40 resolvidas, 11 parciais, 26 abertas), levantadas em 03/11/2026 a partir do código e da documentação. Em 11/11/2026 entraram os sete pacotes de pedidos do fundador, confrontados com o sistema na seção [Pedidos do fundador de 11/11/2026](#pedidos-do-fundador-de-11112026-sete-pacotes), e as seis decisões que eles abrem (PEN-72 a PEN-77). Em 11/11/2026 todas foram conferidas contra o código e os documentos: sete foram encerradas e oito tiveram o texto atualizado. Fonte única no repositório (`docs/pendencias.md`). O que falta decidir, o que está decidido mas não feito e o que está feito, por severidade, está em [decisoes.md](decisoes.md).

Severidade: **Bloqueante** impede operar com cliente real · **Alta** necessária para o primeiro ciclo persistente · **Média** precisa de decisão antes do MVP · **Baixa** higiene.

Para resolver uma pendência: registre a decisão (ADR ou RN), mude o status para `resolvida` com o link e a data, e anote a mudança em `.claude/specs.md`.

| Categoria | Total | Bloqueante | Alta | Média | Baixa |
| --- | --- | --- | --- | --- | --- |
| Negócio e regras | 25 | 7 | 16 | 2 | 0 |
| Produto e escopo | 18 | 1 | 8 | 9 | 0 |
| Técnicas | 14 | 4 | 6 | 2 | 2 |
| Arquitetura e Nexus | 7 | 2 | 3 | 2 | 0 |
| Documentais | 6 | 0 | 2 | 2 | 2 |
| Governança e qualidade | 7 | 2 | 3 | 2 | 0 |

## Negócio e regras operacionais

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-01 | Inventariar a frota real ofertável: veículos próprios e de parceiros, proprietário, documentos e situação | Bloqueante | Fundador / operação | parcial: Jetta 2014 (Economy) e Mercedes C180 (Premium), os dois de parceiros, 03–05/11/2026 (RN-01); de 2 a 5 motoristas no início (06/11/2026, q84, [RN-02.27](../.claude/business-rules/02-reservas.md)); caução de R$ 350 e 200 km por dia nos dois carros (q96, 06/11/2026); o valor do km excedente não foi informado (o limite já pode ser cadastrado sem ele); faltam documentos e situação | Base v1.1 §8.1; Dossiê L.4 |
| PEN-02 | Definir tarifas, período mínimo, limite de km, combustível, atraso e extensão | Bloqueante | Fundador | resolvida em 05/11/2026: valores configuráveis pela operação ([RN-01.5](../.claude/business-rules/01-frota.md)), mínimo de 1 diária, tanque cheio-cheio, com motorista sob consulta, km por dia configurável, 1h de tolerância e depois cobrança por minuto exato (diária ÷ 1440, q89 e CLI, 06/11/2026), extensão pela central ([RN-02.12 a 02.20](../.claude/business-rules/02-reservas.md)); em 06/11/2026: km livre ou limitado acordado na contratação, excedente cobrado (RN-02.18, q82; implementado com odômetro na entrega e na devolução, V7), diária em períodos de 24h (RN-02.24, q67); franquia de km só nas diárias contratadas, somada no período, e limite aceito sem o valor do excedente (q90 e CLI); devolução antecipada e cobrança dos extras decididas pela central (RN-02.28, RN-02.29, q91 e q92) | Base v1.1 §8.2 |
| PEN-03 | Definir requisitos do condutor, caução, cobertura/seguro, responsabilidade, multas e ocorrências | Bloqueante | Fundador + jurídico | parcial: 21 anos, CNH há 1 ano, estrangeiro com habilitação e passaporte, caução por pré-autorização no cartão com valor por veículo (cadastro implementado em 06/11/2026, V7), seguro incluso com franquia, multas repassadas ao condutor principal ([RN-06](../.claude/business-rules/06-condutor-caucao-e-seguro.md), 04–06/11/2026); falta o jurídico confirmar a habilitação estrangeira (PEN-50) | Base v1.1 §8.3 |
| PEN-04 | Decidir a política de reserva: momento do bloqueio, validade da pré-reserva e prazo de pagamento ([ADR-0006](adr/ADR-0006-momento-do-bloqueio-de-reserva.md)) | Bloqueante | Fundador | resolvida em 04/11/2026 ([ADR-0012](adr/ADR-0012-pre-reserva-expira-em-24h.md), RN-02.11): 24h para pagar, depois cancela sozinha e libera o veículo | Dossiê F e L.3 |
| PEN-05 | Definir a ordem entre contrato, assinatura, pagamento e caução e escolher o provedor de assinatura | Bloqueante | Fundador + engenharia | resolvida em 11/11/2026: a ordem está decidida (aceite, pagamento integral na reserva, caução na retirada) e o provedor do pagamento online é a Stripe, para depois do lançamento ([ADR-0013](adr/ADR-0013-pagamento-online-e-moedas.md)); a dúvida sobre assinatura eletrônica segue na PEN-75. Histórico: pagamento integral na reserva, por Pix ou link confirmado pela central no lançamento, aceite dos termos registrado, Stripe para o pagamento online e estorno manual pela central ([ADR-0013](adr/ADR-0013-pagamento-online-e-moedas.md), 04–06/11/2026). Conflito da q71 resolvido em 06/11/2026 pela q88: continua o pagamento integral na reserva, com cancelamento em 24h sem pagamento, e caução na retirada ([RN-02](../.claude/business-rules/02-reservas.md)) | Base v1.1 §5 e §8.5 |
| PEN-06 | Definir cancelamento, multa e estorno, inclusive de locação já paga | Bloqueante | Fundador + financeiro | resolvida em 05/11/2026: transfer segue a LuxuryBus (RN-02.10); aluguel grátis até 48h antes, depois 1 diária, restante estornado ([RN-02.21](../.claude/business-rules/02-reservas.md)). Prazo e meio do estorno dependem do provedor (PEN-05) | Dossiê F |
| PEN-07 | Definir regras de transfer: distribuição, aceite, substituição, atraso, cancelamento, escalada | Alta | Fundador / operação | resolvida em 11/11/2026: regras decididas e módulo implementado; a oferta com prazo saiu em 06/11/2026 (RN-02.30, V14); o que falta (o motorista aceitar por conta própria e ter acesso) segue na PEN-74 e na PEN-14. Histórico: 60 min de espera inclusos, hora extra cobrada, corrida oferecida e o primeiro motorista que aceita leva (RN-02.9, 02.16, 02.17, 04/11/2026); valor da hora configurável (RN-01.5); se ninguém aceitar, a oferta é reaberta com valor maior (RN-02.22, 05/11/2026); a corrida chega pelo WhatsApp da central e pelo painel ou app do motorista (RN-02.27, 06/11/2026); prazo de aceite configurável e acréscimo definido pela central, e desistência volta a oferecer sem penalidade (RN-02.30, RN-02.31, 06/11/2026, q93 a q95); módulo de transfer implementado em 06/11/2026 (oferta com valor e prazo, aceite registrado pela central, desistência, pagamento, conclusão com espera extra, multa de cancelamento; RN-03.14); falta o acesso do motorista ao painel (PEN-14) | Base v1.1 §8.6 |
| PEN-08 | Fixar comissões: percentual, base, elegibilidade, estorno. Conflito ~15% × 20%/15% | Alta | Fundador | resolvida em 11/11/2026: percentuais configuráveis por parceiro, comissão calculada e paga pela central (RN-03.12, RN-03.14); o que falta segue na PEN-76 (comissão por serviço e base do percentual do indicador) e na PEN-65 (acordos com hotéis). Histórico: percentual configurável por categoria (RN-03.4); paga depois do pagamento e do serviço finalizado, então cancelada não gera comissão (RN-03.8, 05/11/2026); percentual por parceiro (RN-07.6) sobre as diárias, saindo da parte do Neon Flow, gerada no encerramento e marcada como paga pela central (RN-03.12, 06/11/2026); falta a elegibilidade (acordo com o hotel, PEN-65) | Revisão 29/09 §14 e §16 |
| PEN-09 | Definir remuneração de motoristas e proprietários/locadoras parceiras | Alta | Fundador | resolvida em 05/11/2026: o Neon Flow recebe, fica com percentual configurável e repassa ao dono (RN-03.7); motorista recebe percentual da corrida (RN-03.9); no transfer e no passeio o dono é o motorista (RN-03.10); repasse calculado no encerramento sobre diárias + atraso + km e marcado como pago (RN-03.13, 06/11/2026) ([RN-03](../.claude/business-rules/03-financeiro-e-comissoes.md)) | Dossiê D |
| PEN-10 | Decidir se a venda de página/bio (~R$400, 5–10 templates) entra no produto | Média | Fundador | aberta | Dossiê D |
| PEN-11 | Aprovar métricas e metas antes de expandir | Alta | Fundador | parcial: indicadores da tela inicial escolhidos em 06/11/2026 (q79): retiradas e devoluções do dia, reservas aguardando pagamento, faturamento do mês, ocupação da frota, repasse a parceiros, comissão de hotéis e avaliação dos clientes; fórmulas em 06/11/2026: faturamento do mês mostra lado a lado as reservas com retirada no mês (com atraso e km) e o que foi pago no mês (q97; exige registrar a data do pagamento), e ocupação = carros alugados hoje ÷ frota (q98), implementadas em 06/11/2026 (`GET /api/dashboard`, `bookings.paid_at` gravado ao marcar o pagamento, V12); conferido em 11/11/2026: o painel já mostra retiradas e devoluções do dia, pré-reservas aguardando pagamento e repasses pendentes (montados no app, `DashboardPage`); faltam as metas e a avaliação dos clientes (exige coletar avaliações) | Revisão 29/09 §14 |
| PEN-12 | Validar mercado: concorrência, preço e demanda atuais | Média | Produto | aberta | Dossiê C |
| PEN-54 | Decidir se o aluguel é com motorista, sem motorista ou os dois | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): os dois | Linha visual LuxuryBus, 03/11/2026 |
| PEN-58 | Definir aeroportos e cidades atendidos (POA, CXJ, Gramado, Canela, outras da Serra) | Alta | Fundador / operação | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): lista base + demais cidades sob demanda | Linha visual LuxuryBus, 03/11/2026 |
| PEN-59 | Decidir se o preço fica público no site (tabela trecho × veículo) ou sob consulta | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): aluguel com preço visível, transfer sob consulta | Linha visual LuxuryBus, 03/11/2026 |
| PEN-61 | Definir as categorias de veículo e se a frota comporta posicionamento premium | Alta | Fundador | resolvida em 03/11/2026 ([RN-01.4](../.claude/business-rules/01-frota.md)): Economy e Premium | Linha visual LuxuryBus, 03/11/2026 |
| PEN-65 | Hotéis parceiros: acordos existentes, autorização de uso de nome e foto, papel (indicador ou ponto de embarque) | Alta | Fundador | parcial: app nasce com template predefinido, área de parceiros depois ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md), 03/11/2026); fotos ilustrativas por enquanto e cadastro do veículo com fotos (implementado em 06/11/2026, bucket do Railway, [ADR-0014](adr/ADR-0014-armazenamento-de-fotos.md)) ([RN-01.7](../.claude/business-rules/01-frota.md), 06/11/2026, q85); faltam acordos e autorização de uso de nome e foto dos hotéis | Linha visual LuxuryBus, 03/11/2026 |
| PEN-66 | Informar o número de WhatsApp da central (canal de reserva do ADR-0011) | Alta | Fundador / operação | resolvida em 05/11/2026: +55 51 99719-2854, configurado em `VITE_WHATSAPP_NUMBER` do `neonflow-app` (Railway `hml`); a landing já abre o WhatsApp da central | Landing, 05/11/2026 |
| PEN-67 | Transfer com carro de parceiro dirigido por outra pessoa: o dono recebe algo pelo uso do carro? Quanto e sobre qual base (preço da corrida, valor fixo)? | Alta | Fundador | aberta: o fundador informou em 06/11/2026 que "nem sempre o motorista do veiculo naquele momento é de fato o proprietário"; o transfer já registra motorista e carro separados (V14), mas só o motorista recebe (RN-03.14) | CLI, 06/11/2026 |
| PEN-70 | Modelo contratual: as minutas V15 tratam o Neon Flow como **intermediador** e o parceiro como locador contratual (04/1.1, 02), com `owner_id`, `contractual_supplier_id` e `payout_beneficiary_id` separados; hoje a central opera a frota dos parceiros (RN-07, fase 1). Qual modelo vale no lançamento? | Alta | Fundador + jurídico | aberta: a Ordem 04 com os dados da reserva espera esta decisão | Pacote V15 de 07/11/2026 (`09_CONTRATOS_V15_REVISAO_JURIDICA_E_DEV_2026-11-07`), 05_ESPECIFICACAO_DEV |
| PEN-71 | Preencher os campos em branco das minutas: razão social, CNPJ, endereço, suporte, política de privacidade, seguradora/apólice, política de cancelamento e percentuais (01, 02, 03 e 04) | Bloqueante | Fundador + jurídico | aberta: as seis minutas v0.4 entram na plataforma como estão, com os campos `[A DEFINIR]`; nenhum aceite real até a aprovação e o preenchimento | Pacote V15 de 07/11/2026 (`09_CONTRATOS_V15_REVISAO_JURIDICA_E_DEV_2026-11-07`), pendências de cada minuta |
| PEN-73 | Pedido vindo do site: bloqueia as datas ao chegar ou só quando a central confirma? O pacote 3 diz "quando a reserva for confirmada, bloqueia aquelas datas"; a regra aprovada bloqueia na criação e solta em 24h sem pagamento ([RN-02.1](../.claude/business-rules/02-reservas.md), [ADR-0006](adr/ADR-0006-momento-do-bloqueio-de-reserva.md), [ADR-0012](adr/ADR-0012-pre-reserva-expira-em-24h.md)) | Alta | Fundador | aberta: hoje só a central cria reserva, então bloquear na criação é seguro; com pedido público, qualquer visitante travaria a frota. Trava o pacote 2 | Pedidos do fundador, 11/11/2026 (pacotes 2 e 3) |
| PEN-74 | Transfer: volta a oferta com aceite e recusa feitos pelo próprio motorista? Em 06/11/2026 o fundador pediu o contrário ("apenas marco ali o motorista que aceitou a corrida") e a oferta com prazo saiu ([RN-02.30](../.claude/business-rules/02-reservas.md), V14) | Alta | Fundador / operação | aberta: hoje a central chama um a um pelo WhatsApp e marca quem aceitou; o motorista não tem acesso ao sistema ([RN-05.6](../.claude/business-rules/05-autorizacao.md), PEN-14). Falta decidir também por onde ele responde (link individual sem login ou área do motorista). Trava o pacote 4 | Pedidos do fundador, 11/11/2026 (pacote 4) |
| PEN-75 | "Assinar digitalmente" o contrato da reserva: é o aceite que já existe (caixa marcada, hash do texto e do quadro, método `link_reserva`) ou assinatura eletrônica com provedor? | Alta | Fundador + jurídico | aberta: a [RN-02.3](../.claude/business-rules/02-reservas.md) aprovou aceite sem assinatura eletrônica; provedor de assinatura segue em PEN-05. Trava o pacote 5 | Pedidos do fundador, 11/11/2026 (pacote 5) |
| PEN-76 | Comissões: (a) a comissão do Neon Flow passa a variar por serviço (locação, transfer, passeio)? Hoje os percentuais são por parceiro ([RN-07.6](../.claude/business-rules/07-parceiros.md)), depois de terem sido por categoria ([RN-03.4](../.claude/business-rules/03-financeiro-e-comissoes.md)); (b) a parte do indicador é um percentual do valor do serviço, como hoje (RN-03.12: sobre as diárias; RN-03.14: sobre o preço da corrida), ou um percentual da comissão da plataforma? | Alta | Fundador | aberta: o texto diz que a participação do indicador "sai de dentro da comissão da plataforma, conforme o percentual combinado"; de onde sai já está implementado, a base do percentual é que ficou ambígua. Nenhum percentual foi informado. Trava o pacote 6 | Pedidos do fundador, 11/11/2026 (pacote 6) |

## Produto e escopo

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-13 | Definir critérios para sair da demonstração e operar com clientes reais | Bloqueante | Fundador + produto | resolvida em 04/11/2026: indispensáveis no dia 1 são landing PT/EN, pedido por WhatsApp/formulário, cadastros de veículos, clientes e motoristas, reserva sem conflito, aceite das condições, checklist com fotos e comissão de hotéis. Pagamento online ficou de fora (ver ADR-0013) | Base v1.1 §8.10 |
| PEN-14 | Especificar a matriz de perfis e permissões | Alta | Produto | parcial: no início só o fundador usa o painel, como admin e central; parceiros (hotéis, proprietários, indicadores) e motoristas depois ([RN-05.6](../.claude/business-rules/05-autorizacao.md), 06/11/2026); gestão de usuários admin/central no painel implementada em 06/11/2026 (RN-05.7); falta a matriz dos perfis de parceiros | Base v1.1 §4 |
| PEN-15 | Especificar a área de clientes (contatos, documentos, preferências, histórico) | Alta | Produto | parcial: portal do cliente desde o início ([RN-04.6](../.claude/business-rules/04-dados-e-integracoes.md), 06/11/2026); cadastro na central com contato, idioma e histórico, ligado às reservas e aos transfers, e anonimização pelo admin (RN-04.7, RN-04.8, 06/11/2026); faltam documentos (PEN-48) e o login do cliente | Dossiê I e J |
| PEN-16 | Especificar cadastro de motoristas e parceiros e edição de frota | Alta | Produto | parcial: cadastro de parceiros pela central com papéis múltiplos, dados básicos e percentuais por parceiro; veículo ligado ao parceiro proprietário ([RN-07](../.claude/business-rules/07-parceiros.md), 06/11/2026); o motorista já entra nas corridas de transfer (V13, V14, 06/11/2026); faltam documentos (PEN-48) e a área do parceiro (PEN-14) | Dossiê I |
| PEN-17 | Especificar indicações: QR, atribuição de origem, resultado ao indicador | Alta | Produto | parcial: todo parceiro indica por link ou QR com código único; o código vai na mensagem do WhatsApp e a central registra "Indicado por" na reserva ([RN-07](../.claude/business-rules/07-parceiros.md), 06/11/2026); o cálculo da comissão foi implementado em 06/11/2026 (RN-03.12, RN-03.14); falta mostrar o resultado ao indicador, na área do parceiro (PEN-14) | Base v1.1 §5; Dossiê J |
| PEN-18 | Detalhar o checklist de entrega e devolução (fotos, combustível, km, avarias) | Média | Produto + operação | parcial: o veículo só volta a ficar disponível com o checklist de devolução aprovado; itens iniciais: combustível, estética, óleo, água, pastilhas, pneus, limpeza interna, documentos, fotos obrigatórias e km, configuráveis pela central e implementados em 06/11/2026 ([RN-01.6](../.claude/business-rules/01-frota.md), 05–06/11/2026); fotos da entrega e da devolução na reserva e o item fotos exigindo foto da devolução (06/11/2026); falta decidir o checklist de retirada (q80 cita a vistoria da Localiza na retirada) | Base v1.1 §5 |
| PEN-19 | Modelar proposta versionada, negociação e mudança de intenção | Média | Produto | aberta | Dossiê E e F |
| PEN-20 | Aprovar a sequência de entrada ([roadmap](produto/roadmap.md)) | Média | Fundador | aberta: em 11/11/2026 o roadmap está defasado (as fases 1 a 3 já foram em boa parte construídas); os sete pacotes do fundador propõem a nova sequência, e falta ele confirmar a ordem | Revisão 29/09 §14 |
| PEN-53 | Decidir o produto principal da home: aluguel, transfer ou os dois | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): aluguel e transfer com o mesmo peso | Linha visual LuxuryBus, 03/11/2026 |
| PEN-55 | Decidir se o shuttle compartilhado (horário fixo, tarifa por pessoa) entra | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): não entra | Linha visual LuxuryBus, 03/11/2026 |
| PEN-56 | Decidir se entram pacotes (transfer + hotel + passeio) | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): mais adiante (passeio + transfer) | Linha visual LuxuryBus, 03/11/2026 |
| PEN-57 | Decidir se haverá convênio corporativo com fatura mensal | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): não entra | Linha visual LuxuryBus, 03/11/2026 |
| PEN-60 | Definir como o cliente fecha a reserva: WhatsApp, formulário para a central ou pagamento online | Alta | Fundador + produto | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): WhatsApp, formulário e pagamento online | Linha visual LuxuryBus, 03/11/2026 |
| PEN-62 | Decidir entre fotos da frota real ou ilustrativas | Média | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): mistas | Linha visual LuxuryBus, 03/11/2026 |
| PEN-63 | Definir idiomas e moedas do site | Média | Fundador | resolvida em 05/11/2026 ([ADR-0013](adr/ADR-0013-pagamento-online-e-moedas.md)): português e inglês; cobrança em reais, dólar só exibido como conversão; conferido em 11/11/2026: a conversão ainda não existe no app, e o site não mostra preço (RN-01.9, ver PEN-59) | Linha visual LuxuryBus, 03/11/2026 |
| PEN-64 | Confirmar se existe logo do Neon Flow para fundo preto e dourado | Média | Fundador | resolvida em 05/11/2026: logo de `nepo/resources/logo.jpg` (anel, estrada e seta, com "neon flow" em Poppins), substituindo o caminho B; SVG redesenhado a partir da imagem, porque o fundador só tem a imagem (06/11/2026, q86). Arquivos em `neonflow-app/public/brand/`; o logo atual é o de `nepo/resources/novo_logo.jpg`, vetorizado nos mesmos arquivos | Linha visual LuxuryBus, 03/11/2026 |
| PEN-72 | Entrada do site só com "aluguel de veículo" e "transfer": para onde vai o aluguel com motorista? Some do site, vira "passeio" (contrato 06, citado no pacote 6) ou fica como opção dentro do aluguel? | Alta | Fundador | aberta: o [ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md) e a [RN-02.15](../.claude/business-rules/02-reservas.md) aprovaram aluguel com e sem motorista; o site tem hoje os três serviços (`ServiceKind` em `neonflow-app/src/modules/landing/model/landing.content.ts`). Trava o pacote 1 | Pedidos do fundador, 11/11/2026 (pacote 1) |
| PEN-77 | Por onde o cliente recebe o retorno da central (confirmado, recusado, ajuste pedido): página de acompanhamento por link, WhatsApp enviado pela central ou e-mail? | Alta | Fundador + produto | aberta: não há provedor de e-mail (PEN-68) nem API de WhatsApp (o site só abre `wa.me`); o link público com token já existe para o aceite (`/public/terms/:token`). Trava o pacote 2 | Pedidos do fundador, 11/11/2026 (pacote 2) |

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
| PEN-30 | Financeiro remove cancelada paga; sem caução, custos, conciliação, repasses | Alta | `finance` (app.js:29); na API, `dashboard_repository.go` e `booking_service.go` (`Cancel`) | parcial: repasses e comissões implementados em 06/11/2026 (RN-03.12 a RN-03.14). Conferido em 11/11/2026 que continua valendo na API: reserva paga e depois cancelada sai do "pago no mês" (RN-03.3 ainda proposta), o cancelamento do aluguel não cobra a diária da RN-02.21, e a caução não é contabilizada (RN-03.1) | Dossiê F |
| PEN-31 | Testes legados fora do CI | Média | `prototipo/tests/legacy/` | resolvida em 06/11/2026: rodam no CI do `neonflow-app` (`.github/workflows/ci.yml`, passo "Testes do protótipo congelado") | 02_CENARIOS |
| PEN-32 | Sem lint, formatação nem CI | Média | raiz | resolvida em 06/11/2026: GitHub Actions em todo PR e push no `main`. API: gofmt, `go vet`, build, testes unitários e de integração com PostgreSQL 16 e cobertura mínima de 80% em `internal/`. App: TypeScript, ESLint, Vitest, build e testes do protótipo | Manifesto |
| PEN-33 | README cita 8 áreas (são 9); agenda com data fixa 27/09/2026 | Baixa | `README.md`, `central` (app.js:24) | resolvida em 11/11/2026 (encerrada sem correção): o protótipo está congelado como evidência ([ADR-0003](adr/ADR-0003-prototipo-estatico-e-evidencia.md)) e não recebe ajustes | Dossiê I |
| PEN-34 | Google Fonts externo; WebMCP nunca testado | Baixa | `style.css:1`, app.js:51; `neonflow-app/index.html` | aberta: conferido em 11/11/2026 que o app também carrega Google Fonts externo (`index.html`: Poppins, Fraunces e Inter); o WebMCP é só do protótipo congelado | Dossiê H |

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
| PEN-43 | Reconfirmar números históricos (35 motoristas, 30 ativos, R$1.500/dia, SLA 60s, 20%/15%, 4–5 visitas/dia) | Alta | Fundador | resolvida em 11/11/2026 (superada): os números não são regras nem metas (RN-03, `produto/metricas.md`) e foram substituídos por respostas do fundador: 2 a 5 motoristas no início (q84), percentuais por parceiro (RN-07.6) e transfer sem prazo de aceite (RN-02.30) | Revisão 29/09 §16 |
| PEN-44 | Reconciliar "Neon Flow" × "Neo Flow" no dossiê comum | Média | Fundador | resolvida em 11/11/2026: o nome está decidido no [ADR-0002](adr/ADR-0002-nome-neon-flow-vertical-unica.md) (aceito); o dossiê comum de 08/10/2026 segue com "Neo Flow" só como referência histórica | 00_LEIA_PRIMEIRO |
| PEN-45 | Ler documentos citados e nunca abertos (Arquitetura_Nexus_OS_Completa.docx, Documento_Recuperacao_Projeto_NeoNoctual_NexusOS.docx, nexus_replit_snapshot.md) | Média | Engenharia | aberta | Dossiê M |
| PEN-46 | Registrar a retificação da regra de comissão no mapa funcional | Baixa | Produto | resolvida em 11/11/2026: registrada em [RN-03.2](../.claude/business-rules/03-financeiro-e-comissoes.md); o protótipo está congelado e não será alterado | Dossiê C.5 |
| PEN-47 | Criar registros de decisão (ADRs) | Baixa | Engenharia | resolvida em 03/11/2026 ([ADR-0001](adr/ADR-0001-registrar-decisoes-em-adr.md)) | — |

## Governança, segurança, LGPD e qualidade

| ID | Pendência | Severidade | Quem resolve | Status | Fonte |
| --- | --- | --- | --- | --- | --- |
| PEN-48 | Tratamento LGPD de CNH, documentos e contatos ([seguranca-e-lgpd.md](seguranca-lgpd.md)) | Bloqueante | Fundador + jurídico | parcial: dados a guardar definidos e o fundador é o encarregado ([RN-04.4 e 04.5](../.claude/business-rules/04-dados-e-integracoes.md), 06/11/2026, q74 e q76); falta o prazo de retenção com o advogado (q75), a base legal e o controle de acesso. Desde 06/11/2026 a API guarda o contato do cliente (RN-04.7) e anonimiza a pedido (RN-04.8), sem dados pessoais em log; documentos continuam fora. Rodada 6 (06/11/2026): guardar também validade, primeira habilitação, categoria e país da CNH (q99), dados e foto (q102) condutor adicional (RN-06.7, q101) e bloqueio da reserva sem motorista de quem não cumpre idade ou habilitação (RN-06.8, q100); o advogado ainda não definiu o prazo (q103) | Dossiê G e H |
| PEN-49 | Isolamento por tenant e por perfil | Bloqueante | Engenharia | aberta | Dossiê K |
| PEN-50 | Parecer jurídico, fiscal e regulatório (locação, transfer, intermediação) | Alta | Fundador + jurídico | resolvida em 04/11/2026 por declaração do fundador: "já consultei, e não precisa". Sem parecer anexado; habilitação estrangeira (RN-06.3): o fundador vai confirmar com o advogado (05/11/2026) | Dossiê C |
| PEN-51 | QA visual, teclado e acessibilidade | Média | QA | aberta | Dossiê I e K |
| PEN-52 | Autorização explícita para implementar (o pacote de 09/10 autoriza só documentação) | Alta | Fundador | resolvida em 03/11/2026 ([ADR-0011](adr/ADR-0011-escopo-de-lancamento-e-canais.md)): landing e reserva autorizadas | Dossiê L; 00_LEIA_PRIMEIRO |
| PEN-68 | Convite da revisora jurídica: hoje o admin cria o usuário `legal_reviewer` com senha (painel ou `create-admin`); o guia pede convite individual sem senha em texto, MFA quando houver e nenhum botão de aprovar valendo em produção antes da homologação | Alta | Engenharia + fundador | aberta: papel e regras de acesso implementados em 08/11/2026 ([RN-08](../.claude/business-rules/08-contratos.md)); falta o convite por e-mail (sem provedor) e o MFA. Dados da Dra. Mônica não são cadastrados por suposição. Em 08/11/2026 a conta dela foi adiada: até ela entrar, o admin também aprova (exceção da RN-08.4), e a exceção tem de sair antes de a advogada assumir | Pacote V15 de 07/11/2026 (`09_CONTRATOS_V15_REVISAO_JURIDICA_E_DEV_2026-11-07`), 06_GUIA_AREA_JURIDICA |
| PEN-69 | Relatório de revisão em PDF e diferença entre versões (critérios 7 e 8 do guia) | Média | Engenharia | aberta: a API devolve texto, hash, eventos e comentários de cada versão; o PDF e a comparação ficam para o painel ou um endpoint de exportação | Pacote V15 de 07/11/2026 (`09_CONTRATOS_V15_REVISAO_JURIDICA_E_DEV_2026-11-07`), 06_GUIA_AREA_JURIDICA |

## Pedidos do fundador de 11/11/2026 (sete pacotes)

Texto entregue pelo fundador em 11/11/2026 (original no fim desta seção), confrontado no mesmo dia com o código da API (`main` em `4848f5f`), do app (`main` em `89ce349` mais o PR 32), as regras em `.claude/business-rules/` e as pendências acima. São pedidos de evolução, não regras aprovadas: o que contradiz uma regra virou pendência (PEN-72 a PEN-77) e nenhuma regra foi alterada.

Situação: **existe** já funciona · **parcial** existe em parte · **falta** não existe.

| Pacote | Resumo | Situação | Travado por | Ordem sugerida |
| --- | --- | --- | --- | --- |
| 1 | Entrada do site mais simples | parcial | PEN-72 | 1º (junto com o 2) |
| 2 | Pedido do site chega à central | falta | PEN-73, PEN-77 | 1º |
| 3 | Ciclo completo da locação | parcial, quase todo pronto | PEN-73, PEN-18, PEN-48 | 2º |
| 4 | Motorista aceita ou recusa a corrida | parcial | PEN-74, PEN-14 | 4º |
| 5 | Contrato aprovado na reserva | parcial | PEN-70, PEN-71, PEN-75 | 3º |
| 6 | Comissões, parceiros e indicações | parcial | PEN-76, PEN-67, PEN-08 | 5º |
| 7 | Preparar a IA | falta | PEN-41, PEN-48 | 6º |

### Pacote 1 — Entrada do site

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| Só dois serviços: aluguel de veículo e transfer | O site oferece três: aluguel, aluguel com motorista e transfer (`ServiceKind`) | falta, depende de PEN-72 |
| Aluguel: escolher o carro e informar retirada e devolução | Formulário único com data inicial e final, sem hora; o carro entra como texto nas observações quando o cliente clica em "Pedir este carro" (`ContactForm.tsx`). A reserva na API já tem data e hora (RN-02.23) | parcial |
| Transfer: origem, destino, data, horário e passageiros | O mesmo formulário, com um campo livre "trajeto"; sem horário nem passageiros. A API do transfer já tem `origin`, `destination`, `pickup_at`, `passengers` e `flight` (V13) | parcial |
| Experiência melhor no celular | Menu compacto do painel feito (app, PRs 30 e 31); o formulário do site não foi revisto | parcial |

### Pacote 2 — Site ligado à central

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| O pedido do site aparece na central com todas as informações | O formulário só monta uma mensagem e abre o WhatsApp (`whatsapp.ts`). Não há rota pública de pedido: `POST /api/bookings` e `POST /api/transfers` exigem login | falta |
| A central confirma, recusa ou pede ajuste | A reserva nasce em "preparação" criada pela central; não existe a situação "solicitada" nem as ações de recusar e pedir ajuste. O transfer exige preço na criação (`price_cents > 0`), e o pedido do site chega sem preço (transfer sob consulta, ADR-0011) | falta, depende de PEN-73 |
| O cliente recebe o retorno | Não há canal: sem e-mail, sem API de WhatsApp | falta, depende de PEN-77 |
| Não cadastrar o mesmo pedido duas vezes | É exatamente o que acontece hoje: a central lê o WhatsApp e digita a reserva | falta |
| Testar o processo completo | Não há o que testar antes da rota de pedido | — |

Além do pedido: rota pública precisa de limite de requisições e proteção contra abuso; o cliente pode ser criado com nome, WhatsApp, e-mail e idioma (RN-04.7), sem documentos (PEN-48); o código de indicação iria direto para o pedido, sem a central ler na mensagem (ponto em aberto da RN-07).

### Pacote 3 — Ciclo completo da locação

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| Cliente escolhe o veículo, informa o período e solicita | Só a central cria a reserva; o site não consulta disponibilidade (`GET /api/availability` exige login) | falta (pacote 2) |
| Sistema verifica a disponibilidade | `GET /api/availability`, com mínimo de diárias por carro (RN-02.32) | existe |
| Reserva confirmada bloqueia as datas | Bloqueio atômico no banco (`EXCLUDE` por `tstzrange`), mas desde a **criação**, não desde a confirmação (RN-02.1) | existe, com divergência: PEN-73 |
| Retirada | Etapa "em locação", com hora real da entrega, odômetro e fotos (V6, V7, V16) | existe |
| Vistoria | Checklist configurável só na **devolução** (RN-01.6); a retirada tem uma conferência única e fotos | parcial: PEN-18 |
| Devolução | Etapa "devolvida", com hora real, atraso por minuto (RN-02.19) e km excedente (RN-02.18) | existe |
| Encerramento | Etapa "encerrada", gera comissão e repasse (RN-03.12, RN-03.13) | existe |
| Tudo registrado na mesma reserva | Histórico (`/bookings/:id/events`), fotos, aceite e repasses ligados à reserva | existe |

Ainda fora do ciclo: multa e estorno do cancelamento (RN-02.21 aprovada, não implementada; PEN-30), extensão do aluguel (RN-02.20), cobrança dos extras (a central decide, RN-02.29), documentos do cliente (PEN-48) e pagamento online (ADR-0013). Uma locação de ponta a ponta não foi executada nesta análise.

### Pacote 4 — Distribuição do transfer

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| Pedido com trajeto, data, horário e valor confirmado | Transfer com origem, destino, `pickup_at`, passageiros, voo e preço | existe |
| A central oferece a corrida aos motoristas cadastrados | Lista de motoristas e proprietários ativos, botão que abre o WhatsApp de cada um e registra o chamado (`/drivers`, `/notify`) | existe |
| Cada motorista aceita ou recusa | Quem marca o aceite é a central (`/accept`); o motorista não tem acesso. Recusa não é registrada | falta: PEN-74, PEN-14 |
| Quando um aceita, a corrida fica com ele | Segundo aceite recusado com a corrida travada (RN-02.17); desistência devolve a corrida (RN-02.31) | existe |
| Depois: distribuição automática por disponibilidade, localização e tempo de resposta | Só "corridas no mesmo dia" por motorista. Sem localização; tempo de resposta não é medido | falta (fase seguinte, como no pedido) |

### Pacote 5 — Contratos nas reservas

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| Ligar os contratos aprovados às reservas | Feito para a Ordem 04: com versão publicada, o link de aceite mostra a Ordem (RN-08.10). **Nenhuma versão está publicada**, então vale o texto de Termos | parcial: PEN-70, PEN-71 |
| Contrato com os dados daquela operação | Os dados vão em um quadro ao lado do texto (cliente, carro, local, período, valor, caução, km), com hash próprio; não entram no corpo do contrato | parcial: PEN-71 |
| Apresentar quando o cliente confirma a locação | A central gera o link à mão em cada reserva (`POST /bookings/:id/terms-link`) | parcial (automático com o pacote 2) |
| Conferir e assinar digitalmente | Aceite com caixa marcada, nome digitado, hash do texto e método; sem assinatura eletrônica (RN-02.3) | parcial: PEN-75 |
| Contrato assinado vinculado à reserva, com documentos e vistorias | `GET /bookings/:id/terms` devolve o aceite; fotos da vistoria na reserva. Sem PDF (PEN-69) e sem documentos do cliente (PEN-48) | parcial |
| Aproveitar a estrutura, sem refazer | Versões, hash, revisão e publicação prontos (RN-08) | existe |

Transfer e passeio ainda não usam as Ordens 05 e 06; os contratos 01, 02 e 03 não entram em nenhum fluxo.

### Pacote 6 — Comissões, parceiros e indicações

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| Veículos de parceiros, motoristas e indicadores (hotéis, restaurantes) | Parceiros com papéis múltiplos e código de indicação por link ou QR (RN-07) | existe |
| A reserva registra quem prestou o serviço | Aluguel: o proprietário pelo veículo (`owner_partner_id`). Transfer: motorista e carro (V14) | existe |
| A reserva registra de onde veio o cliente | "Indicado por" na reserva e no transfer, escolhido pela central. O canal (site, WhatsApp, hotel) não é gravado | parcial |
| A reserva registra os valores de cada participante | Comissão, repasse e motorista nascem no encerramento (`partner_payouts`). A parte do Neon Flow é o que sobra, sem registro próprio | parcial |
| Comissão do Neon Flow varia entre locação, transfer e passeio | Percentuais por parceiro, não por serviço. Passeio não existe como produto | falta: PEN-76, PEN-72 |
| Com indicador, a parte dele sai da comissão da plataforma | Já sai da parte do Neon Flow (RN-03.12, RN-03.14); a base do percentual é o valor do serviço | existe, base a confirmar: PEN-76 |
| Percentuais configuráveis | `commission_bp`, `owner_share_bp` e `driver_share_bp` por parceiro, sem padrão no código | existe |
| Registrar os repasses | Tela Repasses, com marcação de pago, data e quem marcou | existe |

Seguem abertos: o que o dono recebe quando outro motorista usa o carro dele no transfer (PEN-67), elegibilidade da comissão (PEN-08), acordos com hotéis (PEN-65) e a área do parceiro com o resultado das indicações (PEN-17).

### Pacote 7 — Preparar a IA

| O que o fundador pede | Hoje no sistema | Situação |
| --- | --- | --- |
| A IA atende no WhatsApp, entende origem, destino e horário e pergunta o que falta | Nada. O WhatsApp é um link `wa.me`; sem API do WhatsApp Business nem provedor de IA | falta: PEN-41 |
| A IA prepara a solicitação para o sistema | Não há rota para receber solicitação; seria a mesma do pacote 2 | falta (pacote 2) |
| No aluguel, ajudar a encontrar opções disponíveis | `GET /api/availability` existe, só para a central | parcial |
| Não automatizar tudo agora; reduzir o trabalho manual da central | — | — |

O que os outros pacotes já preparam: a rota de pedido do pacote 2, se aceitar pedido incompleto para a central completar, serve também à IA. Guardar conversas exige regra de LGPD (RN-04.1, PEN-48).

<details>
<summary>Texto original do fundador (11/11/2026)</summary>

**PACOTE 1 — Melhorar a entrada do site**

Olha, uma coisa que eu gostaria de melhorar primeiro é a forma como o cliente entra para solicitar um serviço.

Hoje aparece aluguel sem motorista, aluguel com motorista e transfer. Acho que podemos simplificar isso para aluguel de veículo e transfer.

No aluguel, o cliente escolhe o carro e informa quando vai retirar e devolver. No transfer, informa de onde sai, para onde vai, data, horário e passageiros.

Quero deixar essa experiência mais simples e inteligente, principalmente no celular. Dá uma olhada no que já temos e no que podemos aproveitar para melhorar isso.

**PACOTE 2 — Conectar o site com a central**

Outra coisa que eu gostaria de ajustar é a ligação entre o pedido do cliente e a central.

Por exemplo, o cliente entra no site, escolhe um veículo e solicita uma reserva. Quero que esse pedido apareça automaticamente na central, com todas as informações.

A central pode confirmar, recusar ou pedir algum ajuste, e o cliente recebe esse retorno.

A ideia é não precisar cadastrar o mesmo pedido duas vezes. Queria testar esse processo completo para ver o que já está funcionando.

**PACOTE 3 — Finalizar o processo de locação**

Na parte de locação, gostaria que a gente aproveitasse tudo que já foi construído e começasse a fechar o ciclo completo.

O cliente escolhe o veículo, informa o período e solicita a reserva. O sistema verifica a disponibilidade e, quando a reserva for confirmada, bloqueia aquelas datas para não haver outra reserva no mesmo período.

Depois temos retirada, vistoria, devolução e encerramento. Quero que tudo fique registrado na mesma reserva.

Vamos ver o que já funciona e o que falta para conseguir fazer uma locação completa dentro do sistema.

**PACOTE 4 — Evoluir a parte de transfer**

Sobre o transfer, gostei da estrutura que já existe, mas queria melhorar a parte da distribuição das corridas.

Por exemplo, entra um pedido de Porto Alegre para Gramado, com data, horário e valor confirmado.

A central consegue oferecer essa corrida aos motoristas cadastrados. Cada um pode aceitar ou recusar, e quando um aceita, a corrida fica atribuída a ele.

Depois podemos evoluir para o sistema distribuir automaticamente conforme disponibilidade, localização e tempo de resposta.

Primeiro queria ver como podemos fazer esse processo funcionar aproveitando o que já temos.

**PACOTE 5 — Integrar contratos e reservas**

Na parte dos contratos, gostei bastante do controle de versões que você criou.

O próximo passo que eu imagino é conectar os contratos aprovados às reservas.

Por exemplo, quando um cliente confirma uma locação, o sistema apresenta o contrato correspondente, com os dados daquela operação, para ele conferir e assinar digitalmente.

Depois o contrato assinado fica vinculado à reserva, junto dos documentos e das vistorias.

Quero aproveitar a estrutura que você já desenvolveu, sem refazer essa parte. Vamos identificar o que falta para essa integração.

**PACOTE 6 — Comissões, parceiros e indicações**

Outra parte importante é organizar melhor os repasses.

O Neon Flow vai ter veículos de parceiros, motoristas e pessoas ou estabelecimentos que indicam clientes, como hotéis e restaurantes.

Quero que cada reserva registre quem prestou o serviço, de onde veio o cliente e quais valores pertencem a cada participante.

A comissão do Neon Flow pode variar entre locação, transfer e passeios. Quando houver um indicador, a participação dele sai de dentro da comissão da plataforma, conforme o percentual combinado.

Precisamos deixar esses percentuais configuráveis e registrar os repasses. Vamos ver o que já existe nessa parte.

**PACOTE 7 — Começar a preparar a inteligência artificial**

Depois de ajustarmos esses processos, gostaria de começar a preparar o Neon Flow para trabalhar com inteligência artificial.

Por exemplo, alguém chama no WhatsApp e diz que chega ao aeroporto de Porto Alegre amanhã e precisa ir para Canela.

A IA entende o pedido, identifica origem, destino e horário, pergunta o que estiver faltando e prepara a solicitação para o sistema.

O mesmo pode acontecer com aluguel de veículos, ajudando a encontrar opções disponíveis.

Não quero automatizar tudo agora. Quero começar aproveitando a estrutura existente, para que futuramente a IA consiga ajudar a central e reduzir o trabalho manual.

</details>
