# Specs — Changelog de Alterações

---

## Em andamento — Ciclo de reserva persistente

Plano: `docs/plano-reserva-persistente.md`. **Bloqueado** até resolver PEN-01 a PEN-06 (regras de locação), PEN-35/PEN-36 (donos no Nexus/Replit) e PEN-48/PEN-49 (LGPD e isolamento). PEN-52 resolvida: landing e reserva autorizadas (ADR-0011).

Já decidido: o período fica bloqueado desde a criação (RN-02.1) e a pré-reserva não paga em 24h é cancelada sozinha (RN-02.11, ADR-0012, que substitui o ADR-0006). Pagamento integral na reserva, por Pix ou link confirmado pela central no lançamento (ADR-0013). Valores comerciais configuráveis, nunca no código (RN-01.5). O veículo só volta a ficar disponível com o checklist de devolução aprovado (RN-01.6). A stack está definida (ADR-0009).

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
