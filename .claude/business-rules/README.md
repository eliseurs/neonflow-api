# Regras de Negócio — Neon Flow

> Regras da vertical de aluguel e mobilidade. Cada regra cita a fonte e o status.
> Manter sincronizado com o código: ao implementar uma regra, preencha "Onde é aplicada".

## Status

| Status | Significado |
|---|---|
| **aprovada** | Decisão explícita do fundador/time, com data |
| **implementada** | Existe no `neonflow-api`/`neonflow-app` com teste |
| **protótipo** | Existe só em `../neonflow-app/prototipo/dist/` — não aprovada como regra de produto |
| **proposta** | Sugerida em documento, sem aprovação |

**Nenhum valor (tarifa, caução, franquia, comissão, hora de espera) está aprovado.** O fundador decidiu que esses valores são **configuráveis pela operação** (RN-01.5), então não ficam no código. As regras de formato estão aprovadas, como 24h para pagar, 1 diária mínima e cancelamento do transfer. Ver `docs/pendencias.md` (PEN-01 a PEN-12). Não invente valores: sem fonte, abra uma pendência.

## Índice

| Arquivo | Área |
|---|---|
| `01-frota.md` | Veículos, manutenção e tarifa |
| `02-reservas.md` | Ciclo de locação, bloqueio de período, cancelamento e valor |
| `03-financeiro-e-comissoes.md` | Caução, recebimento e comissão por origem |
| `04-dados-e-integracoes.md` | Dados entre verticais e integrações com o Nexus |
| `05-autorizacao.md` | Autenticação e papéis |
| `06-condutor-caucao-e-seguro.md` | Requisitos do condutor, caução e seguro no aluguel sem motorista |
| `07-parceiros.md` | Cadastro de parceiros (proprietário, motorista, indicador) e indicação por link ou QR |
| `08-contratos.md` | Os seis contratos em versões com hash e a revisão jurídica (pacote V15) |

---

## C4 — Nível 1: Contexto do Sistema

```mermaid
C4Context
    title Neon Flow — Contexto (alvo)

    Person(central, "Central / Gestor", "Coordena frota, reservas e exceções")
    Person(cliente, "Cliente / Locatário", "Turista ou morador que aluga ou pede transfer")
    Person(motorista, "Motorista parceiro", "Aceita e executa transfers")
    Person(indicador, "Hotel / Comércio", "Origina solicitações")

    System(neonflow, "Neon Flow", "Oferta, frota, locação, transfer e economia da vertical")
    System_Ext(nexus, "Nexus Core", "Identidade, contexto (CTX), eventos, governança")
    System_Ext(shell, "Shell", "Entrada e continuidade do cliente")
    System_Ext(pagamento, "Provedor de pagamento", "A definir (PEN-05)")
    System_Ext(assinatura, "Assinatura digital", "A definir (PEN-05)")

    Rel(central, neonflow, "Opera frota e reservas")
    Rel(cliente, shell, "Solicita aluguel/transfer")
    Rel(indicador, shell, "Origina solicitação")
    Rel(shell, neonflow, "Encaminha intenção com contexto")
    Rel(motorista, neonflow, "Aceita e conclui transfers")
    Rel(neonflow, nexus, "Identidade, CTX e eventos")
    Rel(neonflow, pagamento, "Cobranças e estornos")
    Rel(neonflow, assinatura, "Contratos")
```

## C4 — Nível 2: Containers

```mermaid
C4Container
    title Neon Flow — Containers

    Person(central, "Central / Gestor")

    Container_Boundary(fe, "Frontend") {
        Container(spa, "neonflow-app", "React + TypeScript + Vite", "Painel da central")
    }
    Container_Boundary(be, "Backend") {
        Container(api, "neonflow-api", "Go 1.25 + Gin", "API REST — auth, frota, reservas")
        Container(migrate, "migrate", "Go CLI", "Runner de migrations SQL")
    }
    ContainerDb(db, "PostgreSQL 16", "Banco relacional", "users, vehicles, bookings")
    Container(proto, "prototipo", "Node + HTML estático", "Demo congelada (commit fdf114ef)")

    Rel(central, spa, "Usa", "HTTPS")
    Rel(spa, api, "Chama", "JSON/HTTPS + JWT")
    Rel(api, db, "Lê/escreve", "pgx")
    Rel(migrate, db, "Aplica migrations")
```

Hoje existem o `neonflow-api` (este repositório, scaffold: health + auth), o `neonflow-app` (repo irmão, scaffold: login + telas vazias) e o `prototipo` (dentro do `neonflow-app`). Frota e reservas ainda não foram implementados.
