# Specs — Changelog de Alterações

---

## Em andamento — Ciclo de reserva persistente

Plano: `docs/plano-reserva-persistente.md`. Núcleo implementado na API em 05/11/2026 (entrada abaixo). Falta: horário e fuso (PEN-29), dados e documentos do cliente (PEN-48), isolamento por tenant (PEN-49), multa e estorno do cancelamento (RN-02.21, PEN-05), estado do veículo com reservas futuras (RN-01.3, PEN-28), limite de km e itens do checklist configuráveis (RN-02.18, RN-01.6) e as telas no `neonflow-app`.

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
