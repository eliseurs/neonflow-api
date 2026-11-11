# plano de reserva persistente — Ciclo de reserva persistente (aluguel assistido)

> **Estado em 05/11/2026:** núcleo implementado na API (`fleet`, `booking`): cadastro da frota, disponibilidade por período, pré-reserva com bloqueio atômico, conferências, etapas, cancelamento, expiração em 24h e histórico. Os cinco cenários abaixo têm teste. Ficaram de fora, por dependerem de pendências: horário e fuso (PEN-29), dados e documentos do cliente (PEN-48), tenant (PEN-49), multa e estorno do cancelamento (RN-02.21, PEN-05) e a tela no `neonflow-app`.

## Objetivo

Primeiro incremento real, sugerido no Dossiê L: um ciclo assistido de reserva com persistência, identidade e disponibilidade atômica, reutilizando a infraestrutura existente do Nexus, com regras explícitas e histórico. Corresponde à fase 1 do [roadmap](produto/roadmap.md).

## Escopo

- **Dentro:** cadastro da frota ofertável, consulta de disponibilidade por período, criação de pré-reserva, conferência de documentos, confirmação, entrega, devolução, encerramento e cancelamento, com histórico de cada transição. Operado pela central.
- **Fora:** autosserviço do cliente, transfer, indicações e comissões, pagamento automático, assinatura automática, IA, marketplace e dispatch autônomo.

## Regras aplicáveis

| RN | Como se aplica | Precisa aprovar antes |
| --- | --- | --- |
| [RN-02.1](../.claude/business-rules/02-reservas.md) | Bloqueio atômico no servidor | PEN-04, PEN-29 |
| [RN-01.1](../.claude/business-rules/01-frota.md) | Bloqueio por manutenção, por período | PEN-01 |
| [RN-02.3](../.claude/business-rules/02-reservas.md) | Evidência documental real | PEN-03, PEN-05 |
| [RN-02.4](../.claude/business-rules/02-reservas.md) | Transições validadas e idempotentes no servidor | PEN-18 |
| [RN-02.5](../.claude/business-rules/02-reservas.md) | Cancelamento preserva recebimento e registra estorno | PEN-06 |
| [RN-02.6](../.claude/business-rules/02-reservas.md) | Valor contratado gravado na reserva (snapshot) | PEN-02 |

## Diferenças em relação ao protótipo

| Item | Protótipo | Classificação |
| --- | --- | --- |
| Persistência | Arrays em memória | gap (PEN-21) |
| Validação | Só no navegador | gap (PEN-22) |
| Autenticação | Inexistente | gap (PEN-23) |
| Concorrência | Sem bloqueio atômico | gap (PEN-24) |
| Momento do bloqueio | Na criação | divergência com a Base v1.1 ([ADR-0006](adr/ADR-0006-momento-do-bloqueio-de-reserva.md)) |
| Preço | Recalculado | inconsistência (PEN-27) |
| Estado do veículo | Ignora reservas futuras | inconsistência (PEN-28) |

## Critérios de aceite (rascunho)

```gherkin
Funcionalidade: Reserva persistente

  Cenário: Duas centrais tentam reservar o mesmo período
    Dado o veículo "V1" livre de 10/10 a 13/10
    Quando duas sessões confirmam reservas para "V1" no mesmo período ao mesmo tempo
    Então apenas uma reserva é criada
    E a outra recebe "período indisponível"

  Cenário: Reserva sobrevive à recarga
    Dado uma reserva criada pela central
    Quando a página é recarregada
    Então a reserva continua com a mesma etapa e o mesmo valor

  Cenário: Valor contratado não muda com a tarifa
    Dado uma reserva confirmada com 3 diárias a R$ X
    Quando a tarifa do veículo muda
    Então o valor da reserva continua 3 × R$ X

  Cenário: Data inválida é rejeitada
    Quando a central informa retirada em 30/02
    Então o sistema rejeita a data

  Cenário: Transição fora de ordem é rejeitada
    Dado uma reserva encerrada
    Quando alguém tenta avançar a etapa
    Então o servidor rejeita e nada muda
```

Valores e prazos (tarifa, TTL da pré-reserva) ficam como variáveis até a aprovação das RNs.

## Bloqueios

> **Defasado em 11/11/2026:** o núcleo foi implementado em 05/11/2026 com autorização do fundador (PEN-52) sem esperar PEN-35, PEN-36 e PEN-49, que seguem abertas. O estado de cada pendência está em [decisoes.md](decisoes.md).

Esta SPEC só sai de `bloqueada` quando estiverem resolvidas: PEN-01 a PEN-06 (regras), PEN-35 e PEN-36 (donos no Nexus/Replit), PEN-40 (stack), PEN-48 e PEN-49 (LGPD e isolamento) e PEN-52 (autorização).
