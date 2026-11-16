# Specs — Changelog de Alterações

---

## Em andamento — Ciclo de reserva persistente

Plano: `docs/plano-reserva-persistente.md`. Núcleo implementado na API em 05/11/2026 (entrada abaixo). Falta: horário e fuso (PEN-29), dados e documentos do cliente (PEN-48), isolamento por tenant (PEN-49), multa e estorno do cancelamento (RN-02.21, PEN-05), estado do veículo com reservas futuras (RN-01.3, PEN-28), limite de km e itens do checklist configuráveis (RN-02.18, RN-01.6) e as telas no `neonflow-app`.

---

## [2026-11-16] — Oferta aberta do transfer (PEN-74)

Decisões do fundador no CLI: a oferta mostra tudo, inclusive o nome do cliente; o motorista desiste pelo próprio app; sem aviso automático por enquanto (a central avisa pelo WhatsApp e a oferta aparece na área do motorista).

- `GET /api/partner/area` traz `offers` para o motorista: corridas em "aguardando motorista" com o valor definido pela central.
- `POST /api/partner/rides/:id/accept` (carro opcional, só um dos dele) e `/release` (só a corrida dele, antes de começar). Os dois usam o `TransferService` da central: mesma trava, mesmo valor, mesmo histórico ("aceito", "desistiu"). Dois aceites simultâneos: um leva, o outro recebe 409 "esta corrida não está mais disponível".
- Testes: serviço, handler, rota e integração com dois motoristas aceitando ao mesmo tempo (repetido 5 vezes), desistência e volta à oferta. Cobertura de `internal/`: 95,2%.
- Regras: RN-02.30 e RN-02.31 com a implementação, RN-07.10; PEN-74 resolvida.

---

## [2026-11-16] — Área do parceiro (RN-07.8 a 07.11)

Decisões do fundador no CLI: o proprietário vê datas e valores sem o cliente; o motorista vê tudo para executar a corrida (com o nome do cliente, sem o telefone); o indicador vê serviço, data e comissão; os valores mostram a base e o percentual.

- Domínio `partnerarea`, `GET /api/partner/area` (`area.parceiro`): o parceiro vem da conta logada no banco (ativo), nunca da requisição. Carros e reservas só para o proprietário, corridas só para o motorista, indicações para todos. Pagamento (`payout`) de cada item com base, percentual, valor e data do pagamento.
- Sem migration: lê `vehicles`, `bookings`, `transfers` e `partner_payouts`.
- Testes: entidade, serviço, handler, rota (cliente e central recebem 403) e integração com dois parceiros (um não vê o do outro; o nome do cliente e as observações do transfer não saem). Cobertura de `internal/`: 95,2%.
- Regras: RN-07.8 a RN-07.11; PEN-14 e PEN-17 atualizadas.

---

## [2026-11-16] — Pedidos do site: o cliente pede, a central responde (RN-02.33 a 02.37)

Decisões do fundador no CLI: só cliente com conta pede; o pedido bloqueia só quando a central confirma (resolve PEN-73); retorno por e-mail, Minha conta e botão de WhatsApp na central (resolve PEN-77); aluguel e transfer; o cliente vê a estimativa; no ajuste, o cliente edita e reenvia.

- `V24_pedidos_do_site`: `site_requests` (aluguel com a estimativa gravada; transfer sem preço) e `site_request_events` (histórico com a mensagem da central). Down testado.
- Domínio `siterequest`: carros livres com estimativa (`QuoteFor`, mesmo cálculo da reserva), criar, listar, ver, reenviar e cancelar (cliente, `area.cliente`, só os próprios pedidos); listar, ver, confirmar, pedir ajuste e recusar (central, `operacao`).
- Confirmar usa os serviços de reserva e de transfer da central: a reserva nasce em preparação (bloqueia e abre as 24 horas). O pedido fica reservado durante a criação (`Claim`) para dois cliques não confirmarem duas vezes; se a reserva falhar (período ocupado), volta a solicitado. Diária, caução ou km diferentes do pedido → 409, a central pede ajuste.
- E-mail ao cliente na confirmação, no ajuste e na recusa, com o link de Minha conta; a falha vai para o log e não desfaz a resposta.
- Código de indicação do pedido vira o parceiro ativo com aquele código.
- Testes: entidade, serviço, handler, rota (403 cruzado entre cliente e central) e integração, inclusive a confirmação de ponta a ponta com os serviços reais. Smoke com a API rodando. Cobertura de `internal/`: 95,2%.
- Regras: RN-02.33 a RN-02.37; RN-01.9 revista para o cliente logado; PEN-73 e PEN-77 resolvidas; pacote 2 com a API pronta.
- Falta no app: pedir logado (carro e período com a estimativa, ou transfer), Minha conta com os pedidos e o reenvio, e a tela da central com confirmar, recusar, pedir ajuste e avisar no WhatsApp.

---

## [2026-11-16] — Login de parceiros e clientes (RN-05.11 a 05.14)

Decisão do fundador no CLI: parceiros de qualquer tipo e clientes entram na aplicação (RN-05.6 revista); o cliente se cadastra sozinho, com o e-mail confirmado pelo Resend; a senha esquecida volta por e-mail; no transfer, oferta aberta (PEN-74, ainda não implementada). Este PR é a base de login; as áreas do parceiro e do cliente vêm nos próximos.

- `V23_contas_parceiros_e_clientes`: papéis `parceiro` e `cliente`; `users.partner_id` e `users.customer_id` (uma conta por cadastro, CHECK por papel); `users.email_verified_at` (equipe e contas antigas já confirmadas); `account_tokens` com links de uso único, só o hash. Down testado.
- `authz`: permissões `area.parceiro` e `area.cliente`; `StaffRole` no lugar de `ValidRole`, para o admin e a CLI criarem só contas da equipe. Em Usuários, lista e edição só da equipe.
- Login: cliente com e-mail sem confirmar recebe 403 depois da senha certa; `/auth/me` traz `partner_id` ou `customer_id`. Parceiro desativado não entra e perde a sessão na hora (`store.ActiveRole`).
- Domínio `account`: cadastro do cliente, confirmação (liga ao cadastro da central com o mesmo e-mail, ou cria um), convite do parceiro pela central, aceite do convite, "esqueci a senha" e nova senha. Respostas iguais com ou sem conta para o e-mail. Validade: 24 h, 7 dias e 1 h.
- `internal/infra/email`: Resend (`resend-go/v3` v3.17.0, aprovada pelo responsável; Go 1.25 mantido). Variáveis `RESEND_API_KEY`, `EMAIL_FROM` e `APP_URL`; sem elas, as rotas de e-mail respondem 503. O log da falha não leva o destinatário.
- Anonimizar o cliente apaga nome e e-mail da conta dele e a desativa.
- Testes: authz, config, envio (servidor local no lugar do Resend), entidade, serviço, handler, rota e integração (fluxos completos, link vencido, uso único, cadastro com dois clientes de mesmo e-mail, parceiro desativado, anonimização). Smoke com a API rodando e um Resend falso: convite, aceite, login do parceiro com 403 na operação, cadastro, login antes e depois de confirmar, nova senha, anonimização. Cobertura de `internal/` com integração: 95,5%.
- Docs: RN-05.2, 05.3 e 05.6 revistas, RN-05.11 a 05.14, RN-04.6, RN-04.8 e RN-02.30; ADR-0016; PEN-14, 15, 74 e 77 atualizadas; PEN-78 (domínio verificado no Resend) e PEN-79 (limite nas rotas públicas) novas; `decisoes.md`, `design.md`, `deploy-railway.md`.
- Falta no app: telas de cadastro, confirmação, convite e nova senha (`/conta/...`), botão "Convidar" no parceiro e o artigo da Central de Ajuda.

---

## [2026-11-16] — Permissões, papel do banco e auditoria das rotas (RN-05.8 a 05.10)

- **Matriz de acesso** em `internal/pkg/authz`: papel → permissões (`operacao`, `contratos`, `clientes.anonimizar`, `termos.publicar`, `usuarios`, `auditoria`). `api.Routes` troca `Roles` por `Permission`, e `middleware.Require(permissão)` substitui o `RoleGuard`. Cada papel continua com o mesmo acesso de antes.
- **Papel do banco:** `AuthMiddleware` recebe `Accounts` e lê `role` de quem está `active` a cada requisição. Desativado → 401 na hora; papel trocado vale sem novo login. Sem banco (testes de rota) vale o papel do token.
- **Auditoria:** middleware `Audit` no router grava em `audit_log` (V22, só inclusão por trigger) as escritas de usuário identificado, o login (o handler identifica o usuário) e os 403. Sem corpo nem query string; falha ao gravar vai para o log e não muda a resposta. `GET /api/audit` (permissão `auditoria`) lista com filtros de usuário, negados e limite.
- `user.permissions` no login e no `/auth/me`, para o app montar menu, telas e layout por perfil.
- Testes: `authz`, middlewares (papel do banco, permissão, auditoria), `api` (rebaixado perde acesso, auditoria do 403), domínio `audit` e integração de `store` e do repository. Cobertura de `internal/` com integração: 95,5%.

---

## [2026-11-11] — Pedidos do fundador (sete pacotes) nas pendências

- O `pendencias.md` da raiz (texto do fundador com sete pacotes de evolução) foi mesclado em `docs/pendencias.md`, que volta a ser o único arquivo: seção "Pedidos do fundador de 11/11/2026", com cada pedido confrontado com o que existe na API e no app, e o texto original no fim.
- Pendências novas, todas de decisão do fundador: PEN-72 (aluguel com motorista na entrada do site), PEN-73 (pedido do site bloqueia ao chegar ou ao confirmar), PEN-74 (motorista aceita a corrida por conta própria), PEN-75 ("assinar digitalmente"), PEN-76 (comissão por serviço e base do percentual do indicador) e PEN-77 (canal de retorno ao cliente). Total: 77.
- Nenhuma regra de negócio nem código mudou.

### Conferência das 77 pendências (mesmo dia)

- Todas conferidas contra o código da API, o app, as regras e os ADRs. Encerradas: PEN-05 (ordem e provedor decididos; assinatura segue na PEN-75), PEN-07 e PEN-08 (o que falta segue em PEN-74, PEN-14, PEN-76 e PEN-65), PEN-33 e PEN-46 (protótipo congelado), PEN-43 (superada) e PEN-44 (ADR-0002). PEN-30 passou a parcial e aponta para a API. Texto atualizado em PEN-11, 16, 17, 20, 34, 63 e 64. Registro: 40 resolvidas, 11 parciais, 26 abertas; 7 bloqueantes em aberto.
- `docs/decisoes.md` (novo): as 77 por estado (falta decidir, falta fazer, decidido e feito) e severidade, mais os pontos em aberto sem PEN. As revisões de severidade (PEN-03, 35, 36, 49, 42) e as reaberturas (PEN-59, PEN-50) ficaram só como proposta na seção 2, sem aplicar.
- Defasados corrigidos: contagens antigas saíram de `README.md`, `docs/README.md` e `.claude/CLAUDE.md`; aviso em `docs/produto/roadmap.md` e em "Bloqueios" de `docs/plano-reserva-persistente.md`; o cabeçalho do registro deixou de citar o documento de leitura do Claude Docs, parado desde 03/11.

---

## [2026-11-08] — Ordem 04 no aceite da reserva (RN-08.10)

- Com uma versão publicada do contrato 04, `GET /public/terms/:token` devolve `order` (versão, título, texto, hash) e o quadro da reserva (local, caução e km além do que já ia), com `quadro_sha256`; `terms` vem nulo. Sem Ordem publicada, nada muda.
- Aceite da Ordem: `contract_version_id`, `sha256`, `quadro_sha256` e `confirm`; o repository trava a versão publicada e a reserva, recalcula o hash do quadro e grava em `contract_acceptances` (V21, append-only) com o snapshot e o método `link_reserva`. Texto ou quadro diferentes do lido → 409. Com a Ordem publicada, aceite dos termos antigos → 409 (recarregar).
- `GET /bookings/:id/terms` devolve o último aceite das duas tabelas, com `document` (`termos` ou `ordem_04`).
- RN-08.10 nova; RN-02.3 marcada como implementada para a Ordem 04. Campos `[A DEFINIR]` e partes da Ordem continuam dependendo de PEN-70 e PEN-71.

---

## [2026-11-08] — Admin revisa contratos até a advogada ter conta

- Decisão no CLI: a conta da advogada fica para depois, e o admin também pede ajustes, aprova e revoga (`reviewers` em `internal/domain/contract/entity.go`). O evento grava `role: admin`, então essas decisões ficam identificáveis. Hash e confirmação expressa continuam obrigatórios; liberar e publicar seguem só com o admin; a central continua sem acesso.
- Exceção temporária à regra do guia V15 ("o admin não pode aprovar em nome da revisora"): RN-08.4, PEN-68 e ADR-0015 atualizados. Tem de sair quando a advogada entrar.

---

## [2026-11-08] — Contratos V15 e área jurídica (revisão por cláusula)

- Fonte: pasta do Drive `09_CONTRATOS_V15_REVISAO_JURIDICA_E_DEV_2026-11-07` (seis minutas v0.4, especificação DEV e guia da área jurídica). O texto das minutas não entra no Git ("não subir ao Git sem autorização do fundador"): vai pela API.
- `V20_contracts`: `contract_documents` (os seis, criados na migration), `contract_versions` (texto + SHA-256), `contract_events` e `contract_comments`; triggers append-only recusam UPDATE e DELETE. Papel `legal_reviewer` em `users.role`.
- Domínio `contract`: situação calculada a partir do último evento (`minuta` → `enviado_para_revisao` → `ajustes_solicitados` | `aprovado_pela_revisora` → `liberacao_administrativa` → `publicado`). A transição trava a versão e confere papel, situação e, na aprovação, o hash lido mais a confirmação expressa. O admin não aprova; a central não acessa. Evento com `clock_timestamp()` para manter a ordem quando há espera pelo lock.
- Testes: tabela de transições, serviço, handler (autor sempre do JWT), rota (401 sem login, 403 para a central) e integração (fluxo completo, hash antigo recusado, append-only). Smoke local com as seis minutas reais: hashes conferem, admin recebe 403 ao aprovar, hash errado 409, aprovação não publica (409). Cobertura de `internal/` em 95,7%.
- Regras: RN-08 nova; RN-05.3 com o papel novo; RN-02.3 adota o aceite do padrão V15 (caixa de marcação, hash e método), que será aplicado na Ordem 04. ADR-0015. Pendências novas: PEN-68 (convite e MFA da revisora), PEN-69 (PDF e diferença entre versões), PEN-70 (intermediador × central operando a frota), PEN-71 (campos em branco das minutas).

---

## [2026-11-06] — Rodada 6 do questionário (cadastro do cliente)

- Respostas registradas: RN-04.4 ganha validade, primeira habilitação, categoria e país da CNH (q99) e confirma a foto (q102); RN-06.7 nova, condutor adicional sem custo com nome, documento e CNH (q101); prazo de retenção ainda não definido (q103), então os documentos seguem fora. RN-06.8 nova (q100, respondida no CLI): o sistema bloqueia a reserva sem motorista de quem não cumpre idade ou tempo de habilitação, quando os documentos entrarem. PEN-48 atualizada.

---

## [2026-11-06] — Cadastro de clientes (fase 1)

- `V15_customers`: tabela `customers` (nome, WhatsApp único quando preenchido, e-mail, idioma, observações, `anonymized_at`); `bookings.customer_id` e `transfers.customer_id`.
- Domínio `customer`: listar com busca (nome, e-mail ou dígitos do WhatsApp), ver com o histórico de reservas e transfers, criar, editar e anonimizar (só admin; apaga nome e contatos também em `client_name` dos pedidos e nas observações do transfer).
- Reserva e transfer aceitam `customer_id`; cliente inexistente responde 400 (distingue pela constraint de FK).
- Nenhum dado do cliente vai para o log (teste dedicado); o logger de requisições já não grava query string.
- Regras RN-04.7 e RN-04.8; PEN-15 e PEN-48 atualizadas. Documentos seguem fora.

---

## [2026-11-06] — Despacho do transfer revisto

- Fundador pediu um fluxo mais direto: a central chama os motoristas um a um e só marca quem aceitou. Sem prazo de oferta (RN-02.30 revista).
- `V14_transfer_despacho`: `transfers.vehicle_id` (carro da corrida); a checagem oferta/prazo sai; histórico antigo em centavos passa a reais.
- `GET /api/transfers/:id/drivers`: motoristas e proprietários ativos (RN-03.10 revista: o motorista nem sempre é o dono), com WhatsApp, corridas no mesmo dia e carros de que é dono. `POST /notify` registra o chamado. `POST /offer` só define ou aumenta o valor. `POST /accept` aceita o valor final e o carro.
- Histórico em reais (`FormatReais`): "R$ 300,00 → R$ 350,00", "Pedro · R$ 350,00". A desistência devolve o valor para a próxima chamada.
- Nova PEN-67: o que o dono recebe quando outro motorista usa o carro dele no transfer.

---

## [2026-11-06] — Módulo de transfer (fase 1, operado pela central)

- `V13_transfers`: tabelas `transfers` e `transfer_events`; `partner_payouts` passa a aceitar `transfer_id` (ou `booking_id`) e o tipo `motorista` (valor fixo, sem percentual).
- Domínio `transfer`: criar (preço sob consulta), abrir ou reabrir oferta com valor e prazo (não diminui), registrar o aceite do primeiro motorista (ativo, com papel de motorista), desistência sem penalidade, marcar pagamento, iniciar, concluir com a espera extra (exige pagamento) e cancelar com multa de 50% dentro de 24h, zero se o voo foi cancelado. Cada ação trava a corrida e grava o histórico.
- Concluir gera, na mesma transação, o pagamento do motorista (oferta + espera) e a comissão do indicador sobre o preço (RN-03.14). `payout` lista as duas origens (`booking_id` ou `transfer_id`, `description` = carro ou "origem → destino").
- Regras: RN-02.9, 02.10, 02.16, 02.17, 02.22, 02.27, 02.30 e 02.31 com onde são aplicadas; RN-03.14 nova; PEN-07.

---

## [2026-11-06] — Indicadores da tela inicial (PEN-11, q97, q98)

- `V12_bookings_paid_at`: `bookings.paid_at`. Marcar a conferência `payment` grava a data (só na primeira vez); desmarcar apaga. Reservas já pagas recebem a data da confirmação (aproximação). Vem na resposta da reserva.
- Domínio `dashboard`: `GET /api/dashboard?month=AAAA-MM` (admin e central). Retiradas no mês: diárias + atraso + km das reservas confirmadas em diante com retirada no mês (Gramado). Pago no mês: diárias das reservas com `paid_at` no mês, sem canceladas. Ocupação: carros com reserva em locação agora ÷ veículos cadastrados.
- Limite: atraso e km cobrados depois da devolução não têm data de pagamento própria; entram no "retiradas no mês", não no "pago no mês".

---

## [2026-11-06] — Comissão do indicador e repasse ao proprietário

- `V11_partner_payouts`: tabela `partner_payouts` (tipo `comissao` ou `repasse`, base, percentual copiado do parceiro, valor, pago em/por), uma por tipo e reserva; preenche as reservas já encerradas.
- No encerramento da reserva (`Transition` para "encerrada"), na mesma transação: comissão do indicador sobre as diárias (`total_cents`) e repasse ao dono do carro parceiro sobre diárias + atraso + km (`total_cents + extra_cents + excess_km_cents`). Carro próprio e reserva sem indicação não geram nada.
- Domínio `payout`: `GET /api/payouts` (filtros tipo, situação, parceiro) e `POST /api/payouts/:id/pay` (admin e central). Percentual nulo no encerramento: a definir, e o pagamento usa o percentual do parceiro naquele momento (409 se continuar nulo).
- Regras RN-03.12 e RN-03.13; RN-03.2 e RN-03.4 atualizadas; PEN-08 e PEN-09.

---

## [2026-11-06] — Atraso por minuto, km controlado e rodada 5 do questionário

- `V10_atraso_e_km`: `bookings.extra_minutes` substitui `extra_days` (reservas antigas: 1 diária = 1440 minutos, valor igual). As checagens do plano de km passam a aceitar limite sem valor do excedente; valor sem limite continua recusado.
- `LateMinutes` e `LateCents` substituem `LateDays`: passada 1 hora de tolerância, cada minuto completo desde o combinado custa a diária ÷ 1440, arredondada ao centavo (RN-02.19). A resposta da API traz `extra_minutes` no lugar de `extra_days`.
- Km controlado (RN-02.18): franquia somada no período, só nas diárias contratadas (q90); com limite e sem valor do km, a devolução mede `excess_km` e deixa `excess_km_cents` em 0 para a central definir (RN-02.29). `ErrKmPlanIncomplete` agora é só "valor sem limite".
- Respostas q87–q98 registradas: conflito da q71 resolvido pela q88 (pagamento na reserva), RN-02.28 a RN-02.31 (devolução antecipada, extras e transfer), q96 (caução e km dos carros), q97 e q98 (fórmulas da tela inicial) em PEN-01, 02, 05, 07 e 11.

---

## [2026-11-06] — Cadastro de parceiros e indicação (fase 1)

- `V9_partners`: tabela `partners` (papéis proprietário, motorista e indicador, ao menos um; dados básicos; `referral_code` único; percentuais por parceiro em pontos-base, nulos = a definir), `vehicles.owner_partner_id` e `bookings.referred_by_partner_id` (FKs opcionais).
- Domínio `partner`: `GET/POST /api/partners`, `GET/PUT /api/partners/:id`, `GET /api/partners/code/:code` (admin e central). Código de 6 caracteres sem ambíguos, sorteado de novo em colisão (até 5 tentativas). Repasse ao proprietário só vale para proprietário, percentual do motorista só para motorista.
- Frota e reservas aceitam `owner_partner_id` e `referred_by_partner_id`; parceiro inexistente responde 400.
- Regras: novo `07-parceiros.md` (RN-07.1 a RN-07.7); RN-03.4 passa a ser por parceiro; PEN-16 e PEN-17 parciais. Comissão e repasse calculados ficam para a fase 2.

---

## [2026-11-06] — Gestão de usuários no painel

- `V8_users_active`: `users.active` (padrão true). Login (`FindByEmail`) e `/auth/me` (`FindByID`) só veem usuários ativos; `/auth/me` passa a responder 401 (antes 404) para usuário inexistente ou desativado, e o painel volta ao login.
- Domínio `user`: `GET/POST /api/users`, `PUT /api/users/:id`, `POST /api/users/:id/password` (só admin) e `POST /api/auth/password` (qualquer usuário, com a senha atual). Mesma validação da CLI (senha de 10+ caracteres, papel admin/central). O admin não pode se desativar nem tirar o próprio papel de admin (`ErrSelfLockout`, 409).
- RN-05.2 atualizada e RN-05.7 nova. Limite: o token de um desativado vale até expirar (12 h).
- Testes: entidade, serviço (criação com hash, autobloqueio, senhas), handler, rota e integração (e-mail único, ativo mantido, senha). Login de inativo recusado na integração de auth. Cobertura em 95,9%.

---

## [2026-11-06] — Caução, plano de km e odômetro

- `V7_caucao_e_km`: no veículo, `deposit_cents`, `km_limit_per_day` e `extra_km_cents` (opcionais; limite e valor juntos). Na reserva, cópia dos três na criação e `delivered_km`, `returned_km`, `excess_km`, `excess_km_cents`.
- Frota: `VehicleRequest` aceita os três campos; validação (`ErrInvalidDeposit`, `ErrInvalidKmPlan`, `ErrKmPlanIncomplete`). A disponibilidade devolve caução e plano de km do veículo.
- Reserva: `km_plan` (livre/limitado; padrão: limitado se o veículo tiver limite; `ErrKmPlanUnavailable` se pedir limitado sem limite). No avanço, `km` (odômetro) é obrigatório no plano limitado; na devolução, `ExcessKm` = rodados − limite × (diárias + atraso), cobrado pelo valor copiado (RN-02.18). Recusa km negativo e km da devolução menor que o da entrega.
- ADR-0014 (proposta): fotos dos veículos no Cloudflare R2 com URL assinada. Precisa de aprovação (conta, segredos e dependência nova).
- Testes: validação, `ExcessKm`, serviço (odômetro obrigatório, excedente, entregue sem odômetro), handler e integração (plano padrão, recusa, cópia, odômetro). Cobertura em 95,5%.

---

## [2026-11-06] — Hora real da entrega e da devolução e atraso

- `V6_bookings_hora_real`: `delivered_at`, `returned_at`, `extra_days`, `extra_cents` e CHECK de devolução depois da entrega.
- `POST /bookings/:id/advance` aceita `{"at": "AAAA-MM-DDTHH:MM"}` (padrão: agora). Na entrega grava `delivered_at`; na devolução grava `returned_at` e calcula o atraso: 1 diária por bloco de 24 horas iniciado depois de 1 hora de tolerância (`booking.LateDays`, RN-02.19). Hora no futuro (`ErrFutureTime`) e devolução antes da entrega (`ErrReturnBeforeDelivery`) dão 400.
- `Repository.Transition` recebe `booking.Record` (campos nulos não mudam a reserva). `BookingService` ganhou relógio injetável (`now`).
- Entrega recusada com 409 se o carro ainda está "em locação" em outra reserva (`ErrVehicleInUse`), além do checklist pendente (RN-01.6).
- Testes: `LateDays`, hora real e atraso no serviço, corpo do avanço no handler, gravação e CHECK na integração. Cobertura em 95,3%.

---

## [2026-11-06] — Reserva com horário

- `V5_bookings_horario`: `start_at` e `end_at` (`TIMESTAMPTZ`) no lugar de `start_date` e `end_date`; bloqueio por `tstzrange` (RN-02.2, RN-02.23). Reservas existentes viram 09:00 de Gramado, sem mudar as diárias. Down testado (volta para datas).
- `booking.ParseInstant`: RFC 3339, `AAAA-MM-DDTHH:MM` no horário de Gramado ou só a data (09:00, compatibilidade). `Period.Days`: blocos de 24 horas, para cima, mínimo 1 (RN-02.24).
- API devolve `start_at`/`end_at` no fuso de Gramado e mantém `start_date`/`end_date`; aceita `start_at`/`end_at` (ou as datas antigas) em `POST /bookings` e em `/availability`. `next_booking` da frota também traz a hora, e "reservado" vale até a devolução combinada.
- Novo `internal/pkg/timezone` (America/Sao_Paulo, `time/tzdata` embutido).
- Testes: período, diárias, `ParseInstant`, sobreposição por hora, saída no mesmo dia e fuso. Cobertura em 95,1%.
- Falta: hora real da entrega e da devolução, para a tolerância e a diária extra do atraso (RN-02.19).

---

## [2026-11-06] — Checklist de devolução e estado do veículo

- `V4_checklist`: tabela `checklist_items` com a lista inicial do fundador (RN-01.6, q56 e q80). Chave em minúsculas sem acento, única e imutável.
- Domínio `checklist`: `GET/POST /api/checklist-items`, `PUT /api/checklist-items/:id` (admin e central). A chave sai do nome (`KeyFromLabel`) e não pode repetir uma conferência fixa da reserva (`booking.ReservedCheckKeys`).
- Reservas: em "devolvida", `ValidateChecks` e `NextStage` recebem os itens ativos (`Repository.ReturnChecklist`), então encerrar exige `issues` e todo o checklist. `Transition` para "em_locacao" recusa com 409 (`ErrAwaitingChecklist`) se o veículo tiver outra reserva em "devolvida" (RN-01.6, RN-02.25).
- Frota: `GET /api/vehicles` e `/:id` trazem `status` (manutencao, em_locacao, aguardando_checklist, reservado, livre) e `next_booking` (RN-01.3). "Hoje" em `America/Sao_Paulo`. Create e Update releem o veículo.
- Testes: entidade, serviço, handler e rota do checklist; integração do checklist, do bloqueio da entrega e dos estados da frota. Cobertura de `internal/` em 95,2%.

---

## [2026-11-06] — CI no GitHub Actions

- `.github/workflows/ci.yml`: em todo PR e push no `main`, roda `make lint` (gofmt + `go vet`, inclusive com a tag `integration`), build de `cmd/...`, `make test-integration` com PostgreSQL 16 como serviço do job (`TEST_DATABASE_URL` aponta para ele) e `make cover`, que falha abaixo de 80% em `internal/`.
- O `neonflow-app` ganhou workflow equivalente (TypeScript, ESLint, Vitest, build e testes do protótipo).
- PEN-31 e PEN-32 resolvidas. Pendências: 33 resolvidas, 12 parciais, 21 abertas.

---

## [2026-11-06] — Rodada 4 do questionário do fundador (q66–q86)

- Respostas registradas: RN-01.3 aprovada ("Reservado" com a próxima reserva), RN-01.6 com os itens extras do checklist, RN-01.7 (fotos no cadastro do veículo), RN-02.18 (km livre ou limitado, excedente cobrado), RN-02.23 a RN-02.27 (horário livre, diária de 24h, saída no mesmo dia com checklist, central 24h, transfer pelo WhatsApp e pelo painel do motorista), RN-03.11 (estorno manual), RN-04.4 a RN-04.6 (dados do cliente, encarregado da LGPD, portal do cliente), RN-05.6 (só o fundador usa o painel no início), RN-06.4 (caução por veículo). ADR-0013: Stripe, estorno manual, caução por veículo.
- **Conflito registrado, não resolvido:** q71 (pagamento e caução na retirada) contra RN-02.14/ADR-0013 (pagamento integral ao reservar). Vale o implementado até o fundador confirmar.
- Suposição técnica: fuso `America/Sao_Paulo`.
- `docs/pendencias.md`: PEN-28 e PEN-29 resolvidas; PEN-11, PEN-14, PEN-15 e PEN-48 parciais; PEN-01, 02, 03, 05, 07, 18, 64 e 65 atualizadas. Totais: 31 resolvidas, 14 parciais, 21 abertas.

---

## [2026-11-05] — WhatsApp da central (PEN-66)

- PEN-66 resolvida: número da central +55 51 99719-2854, definido em `VITE_WHATSAPP_NUMBER` do `neonflow-app` no Railway `hml`. O botão flutuante, o rodapé e o formulário da landing já abrem o WhatsApp.
- `docs/pendencias.md`: totais atualizados (29 resolvidas, 10 parciais, 27 abertas).

---

## [2026-11-05] — Landing no neonflow-app

- Landing pública em `/` no `neonflow-app`, só com regras aprovadas (ADR-0011, ADR-0013, RN-02, RN-06), sem preço, foto ou hotel parceiro.
- `docs/pendencias.md`: PEN-66 (número de WhatsApp da central), 66 pendências.

---

## [2026-11-05] — Frota e reservas na API

Decisões do time nesta rodada: diária no veículo (RN-01.2), frota e reservas juntas, com suposições onde as pendências seguem abertas.

- **Migrations**: `V2_vehicles` (placa única, categoria economy/premium, câmbio, diária > 0, manutenção) e `V3_bookings` (`btree_gist`, `EXCLUDE` por veículo e `daterange` nas etapas ativas, cópia da diária e do total, `checks` JSONB, `expires_at`, `created_by`) + `booking_events` (histórico de etapas).
- **Domínio `fleet`**: `GET/POST /api/vehicles`, `GET/PUT /api/vehicles/:id`, para admin e central. Placa normalizada (sem hífen, maiúsculas, formato antigo ou Mercosul).
- **Domínio `booking`**: `GET /api/availability`, `GET/POST /api/bookings`, `GET /api/bookings/:id`, `GET /api/bookings/:id/events`, `PATCH /api/bookings/:id/checks`, `POST /api/bookings/:id/advance`, `POST /api/bookings/:id/cancel`.
  - Bloqueio atômico no banco: sobreposição vira 409 "período indisponível" (RN-02.1, RN-02.2, PEN-24).
  - Etapas preparacao → confirmada → em_locacao → devolvida → encerrada, cada uma com suas conferências (`cnh`, `terms`, `deposit`, `payment` / `delivery` / `return` / `issues`). Avanço e cancelamento só gravam se a etapa ainda for a lida (dois cliques não avançam duas vezes).
  - Pré-reserva sem `payment` vence em 24h (RN-02.11): a expiração roda no início de cada operação de reserva, na mesma transação da criação, e grava evento com `user_id` nulo.
- **Suposições** (registradas nas RNs e em `pendencias.md`): só datas, sem horário (PEN-29); do cliente só o nome, CNH como conferência (PEN-48); operação única (PEN-49); cancelamento sem multa nem estorno (RN-02.21).
- **Testes**: unitários de entidade, service, handler e rota; integração dos dois repositories, incluindo 8 criações simultâneas no mesmo período (só uma passa), expiração e cópia da diária. Cobertura de `internal/`: 94,9% com integração, 64,6% só unitários. Smoke de ponta a ponta com a API rodando.
- **Regras**: RN-01.1, 01.2, 01.4, 02.1, 02.2, 02.3 (como conferência), 02.4, 02.5, 02.6, 02.7, 02.11 e 02.12 marcadas `implementada`.
- **Pendências**: PEN-21 a PEN-27 resolvidas na API (28 resolvidas, 10 parciais, 27 abertas).
- **Railway**: `APP_ENV` do serviço passou de `production` para `hml`. Domínios agora são `neonflow-api-hml` e `neonflow-app-hml`; docs atualizados (`.claude/CLAUDE.md`, `design.md`, `docs/deploy-railway.md` e o do `neonflow-app`).

---

## [2026-11-05] — Ambiente hml no Railway

O ambiente do Railway foi renomeado de `production` para `hml`, e o serviço passou a usar `APP_ENV=hml`. Como a API só exigia `JWT_SECRET` e `DATABASE_URL` com `APP_ENV=production`, em `hml` ela subiria com os defaults locais, inclusive o segredo JWT de desenvolvimento.

- `internal/config/settings.go`: `IsProduction()` virou `IsDeployed()`, verdadeiro para todo `APP_ENV` diferente de `local`. Constante `EnvHml`. Mensagens de erro dizem "fora do ambiente local".
- `cmd/api`, `cmd/create-admin` e `internal/infra/api/http.go` usam `IsDeployed()` (logs em JSON e Gin em modo release em hml).
- Testes: `TestIsDeployed` e `TestValidate_Deployed` (hml, production e um nome qualquer; o segredo de desenvolvimento é recusado).
- Docs: `.claude/CLAUDE.md`, `rules.md`, `design.md` e `docs/deploy-railway.md`. Os domínios `*-production.up.railway.app` não mudaram.

---

## [2026-11-05] — Logo escolhido

- Caminho B (serra e estrada num círculo) escolhido entre três propostas no canvas "Logo Neon Flow" (Claude Design). Versões: principal, horizontal, fundo claro (dourado #7A5C26), ícone de app e favicon simplificado (16 px sem a estrada).
- `docs/pendencias.md`: PEN-64 resolvida (21 resolvidas, 10 parciais, 34 abertas). Falta exportar os arquivos para o `neonflow-app`.

---

## [2026-11-05] — Respostas do fundador, rodada 3

- 16 de 16 respondidas (q50–q65). Os 4 conflitos da rodada 2 foram resolvidos: 21 anos (RN-06.1), o Neon Flow recebe e repassa ao dono (RN-03.7), comissão depois do pagamento e do serviço finalizado (RN-03.8), Pix ou link confirmado pela central no lançamento (RN-02.14, ADR-0013).
- Regras novas:
  - RN-01.6: checklist de devolução libera o veículo; itens configuráveis (observação do fundador na q56).
  - RN-02.18 a RN-02.22: km por dia configurável, atraso com 1h de tolerância e depois 1 diária, extensão pela central, cancelamento do aluguel (grátis até 48h, depois 1 diária) e oferta do transfer reaberta com valor maior.
  - RN-03.10: dono e motorista são a mesma pessoa só no transfer e no passeio.
  - RN-06.6: multas repassadas ao condutor principal.
- ADR-0013 atualizado: meio de pagamento no lançamento; moeda real e dólar, com o dólar só exibido.
- Frota: Jetta e C180 são de parceiros. Logo: símbolo ligado à Serra Gaúcha. Habilitação estrangeira: o fundador vai confirmar com o advogado.
- `docs/pendencias.md`: 20 resolvidas, 11 parciais, 34 abertas. PEN-02, PEN-06, PEN-09 e PEN-63 resolvidas; PEN-18 passou a parcial.

---

## [2026-11-04] — Respostas do fundador, rodada 2

- 24 de 24 respondidas (q26–q49). Rodada 1 sem alterações.
- ADRs novos: ADR-0012 (pré-reserva expira em 24h; ADR-0006 marcado como substituído) e ADR-0013 (pagamento integral online, aceite dos termos registrado, real + moeda estrangeira pela cotação).
- `.claude/business-rules/06-condutor-caucao-e-seguro.md` (novo): RN-06.1 a RN-06.5.
- Regras novas ou alteradas:
  - RN-01.5: valores configuráveis.
  - RN-02.1: expira.
  - RN-02.3: aceite em vez de contrato.
  - RN-02.10: só transfer.
  - RN-02.11 a RN-02.17: expiração, 1 diária, combustível, pagamento online, com motorista sob consulta, hora extra de espera, oferta aos motoristas.
  - RN-03.7 a RN-03.9: percentual do Neon Flow, momento do repasse, percentual do motorista.
- Conflitos registrados como `proposta`:
  - Idade de 18 × 21 anos (RN-06.1).
  - Momento da comissão (RN-03.8).
  - Quem recebe primeiro, o dono ou o Neon Flow (q43 × q44).
  - Pagamento online obrigatório (q37), mas fora da lista do dia 1 (q47).
- `docs/pendencias.md`: 16 resolvidas, 14 parciais, 35 abertas. PEN-04, PEN-13 e PEN-50 resolvidas.
- Rodada 3 publicada no artifact (versão 3, 05/11/2026): q50–q65, com os 4 conflitos (idade, quem recebe primeiro, momento da comissão, pagamento no dia 1) e os pontos sem resposta (cancelamento do aluguel, km, atraso, extensão, multas, corrida sem motorista, motorista × dono, dono dos carros, moeda, habilitação estrangeira, estilo do logo).

Próximos domínios: `fleet` (veículos, RN-01) e `booking` (reservas, RN-02), com a constraint `EXCLUDE` descrita em `design.md`.

---

## [2026-11-03] — Respostas do fundador (questionário)

- Fonte: artifact "Decisões do Neon Flow", coleção `respostas`, 25 de 25 respondidas.
- `docs/adr/ADR-0011-escopo-de-lancamento-e-canais.md` (novo): home com aluguel e transfer, aluguel com e sem motorista, sem shuttle nem convênio, pacotes depois, aluguel com preço e transfer sob consulta, WhatsApp + formulário + pagamento online, PT/EN, logo a criar.
- Regras novas: RN-01.4 (categorias Economy e Premium), RN-02.8 (24h para pagar), RN-02.9 (atraso de voo), RN-03.4 (comissão configurável por categoria), RN-03.5 e RN-03.6 (custos e repasse ao dono do veículo). RN-02.3 anota que não há assinatura eletrônica por enquanto.
- `docs/pendencias.md`: 13 resolvidas, 13 parciais, 39 abertas. Continuam abertas por falta de resposta objetiva: PEN-02, PEN-03, PEN-13 e PEN-50.
- RN-02.10 (cancelamento grátis até 24h, 50% depois, sem cobrança se o voo for cancelado) e os 60 min de espera da RN-02.9 vieram do FAQ da home de luxurybus.ch. Os termos (`/termini`) não têm números.
- Conflito registrado em `02-reservas.md`: o bloqueio não expira (RN-02.1), mas o pagamento vence em 24h (RN-02.8).
- Rodada 2 publicada no mesmo artifact (versão 2): 24 perguntas, q26–q49, sobre PEN-01, 02, 03, 04, 05, 06, 07, 08, 09, 13, 50, 63 e 64.

---

## [2026-11-03] — Referências visuais (local)

- `.gitignore`: ignora `docs/referencias-visuais/` (capturas de sites de terceiros e análise de padrões, só na máquina local).
- Linha visual definida pelo fundador: luxurybus.ch (Luxury Drive). Preto, dourado `oklch(74% .09 80)`, Fraunces + Inter. Detalhes no README local da pasta. O aluguel segue o mesmo design.
- `docs/pendencias.md`: PEN-53 a PEN-65, dúvidas abertas pela linha visual (posicionamento, operação e preço, marca). Total 65 pendências.

---

## [2026-11-03] — Migrations no start do container

O primeiro deploy subiu sem tabelas: login respondia 500 (`relation "users" does not exist`). Causa: serviços Railway novos ignoram o `railway.json` (config-as-code descontinuado), então o `preDeployCommand: ./migrate up` nunca rodou. Correção: `CMD ["sh","-c","./migrate up && exec ./api"]` no Dockerfile; `preDeployCommand` removido do `railway.json` e o `command` duplicado do `docker-compose.yml`. Healthcheck do Railway fica pendente de configuração no painel.

## [2026-11-03] — Publicação no Railway

- Serviço `neonflow-api` criado a partir de `eliseurs/neonflow-api`, com plugin `Postgres` (`DATABASE_URL` por referência), `APP_ENV=production`, `JWT_SECRET` gerado, `ALLOWED_ORIGINS` = frontend. URL: https://neonflow-api-production.up.railway.app.
- Frontend em https://neonflow-app-production.up.railway.app; demo em https://neonflow-prototipo-production.up.railway.app.

## [2026-11-03] — Scaffold da API e separação em dois repositórios

Repositório criado (`eliseurs/neonflow-api`), separado do frontend (`eliseurs/neonflow-app`), seguindo cognia-api (ADR-0009) e o formato `.claude/` (ADR-0010).

- **Infra**: `internal/config` (defaults locais, validação de produção), `internal/infra/api` (Gin, `Routes`, `Respond`, healthcheck com ping do banco, graceful shutdown), `internal/infra/db`, `internal/infra/migrator` (V{N}, transação por migration, valida sequência).
- **pkg**: `security` (JWT HS256 com issuer, expiração obrigatória e só HS256; bcrypt), `middleware` (auth, role guard, CORS por allowlist), `logger` (slog; não loga query string nem corpo).
- **Domínio `authentication`**: `POST /api/auth/login`, `GET /api/auth/me`; mesma mensagem para e-mail inexistente e senha errada, com comparação bcrypt fictícia para igualar o tempo (RN-05.5).
- **CLIs**: `cmd/migrate` (up|down|status), `cmd/create-admin` (lê `ADMIN_*` do ambiente para a senha não aparecer no `ps`).
- **Migration V1**: `users` com `CHECK` de papel e índice único em `lower(email)`.
- **Deploy**: Dockerfile multi-stage (usuário não-root, healthcheck), `railway.json`.
- **Local**: `docker-compose.yml` (PostgreSQL 16 + API) e `devops/initdb` cria `neonflow_test`.
- **Testes**: unitários em todos os pacotes de `internal/` (cobertura 70,3% só com unitários). Integração para repository, migrator e db com build tag `integration` — **ainda não executados** (sem banco local nesta rodada).
- **Docs**: regras de negócio migradas da wiki (`RN-0001…0009` → `RN-01.1…04.1`) para `.claude/business-rules/`; ADRs, pendências, produto, integrações, LGPD, fontes e plano em `docs/`. ADR-0003 e ADR-0006 aceitos; ADR-0009 e ADR-0010 criados; ADR-0007 substituído. PEN-40 resolvida, PEN-04 parcial.

## [2026-11-03] — Demo no Railway (repositório neonflow-app)

Protótipo publicado em https://neonflow-app-production.up.railway.app (serviço `neonflow-app`, projeto Railway `nexus-os`), com `server.mjs` sem dependências. Basic Auth foi ativado e depois removido a pedido do time: o acesso é público.

## [2026-11-03] — Wiki docs-as-code e pendências

Levantamento de 52 pendências (15 bloqueantes) cruzando o código do protótipo com o dossiê de 09/10. Primeira wiki (ADR-0007), depois migrada para este formato.

## [2026-10-09] — Pacote de consolidação v1 (Drive)

Dossiê A–M, 12 testes de auditoria e manifesto com hashes. Fonte: `docs/fontes.md`.

## [2026-09-29] — Protótipo Sites v2 (`fdf114ef`)

Capítulos 13–16 (revisão estratégica) e área de estratégia.

## [2026-09-27] — Protótipo Sites v1 (`b53b31f0`)

Primeira versão do site e Base funcional v1.1.
