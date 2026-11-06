# RN-06 — Condutor, Caução e Seguro

Vale para o **aluguel sem motorista**. No aluguel com motorista e no transfer, quem dirige é o motorista parceiro.

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-06.1 | Idade mínima do condutor: **21 anos** | aprovada (05/11/2026, fundador, q50) | — | PEN-03 |
| RN-06.2 | O condutor precisa ter habilitação há **pelo menos 1 ano** | aprovada (04/11/2026, fundador) | — | PEN-03 |
| RN-06.3 | Turista estrangeiro pode alugar com a **habilitação do país de origem e o passaporte** | aprovada (04/11/2026, fundador) | — | PEN-03, PEN-50 |
| RN-06.4 | A caução é uma **pré-autorização no cartão de crédito**, liberada quando o carro volta em ordem. O valor é **por veículo**, configurável pela central | implementada (06/11/2026; aprovada em 04/11/2026, valor por veículo em 06/11, q73): `deposit_cents` no veículo, opcional (nulo = a definir), copiado para a reserva na criação. A pré-autorização no cartão continua manual (conferência `deposit`) | `V7_caucao_e_km.up.sql` | PEN-03 |
| RN-06.5 | O **seguro está incluso na diária**. Em caso de dano, o cliente paga só a franquia | aprovada (04/11/2026, fundador); seguradora e valor da franquia não informados | — | PEN-03 |
| RN-06.6 | Multas de trânsito durante o aluguel são repassadas ao **condutor principal**, quem está com o veículo | aprovada (05/11/2026, fundador, q58) | — | PEN-03 |

**Fontes:** questionário "Decisões do Neon Flow", q32 a q36 (04/11/2026), q50 e q58 (05/11/2026), q73 (06/11/2026).

## Pontos em aberto

- **RN-06.1:** seguradoras costumam ter idade mínima própria para o condutor; conferir com a apólice.
- **RN-06.3:** a habilitação estrangeira só vale no Brasil em certas condições (país de origem, prazo de estadia, às vezes tradução ou Permissão Internacional). O fundador **vai confirmar com o advogado** (05/11/2026, q64; PEN-50). Até lá, não liberar o aluguel sem motorista para estrangeiros sem CNH brasileira.
- Valor da caução: por veículo (q73), cadastrado com o veículo como a diária (RN-01.2). A franquia não foi perguntada; pelo padrão das respostas, também configurável.
- Momento da caução: **na retirada** (q71). O pagamento na retirada, marcado na mesma pergunta, foi descartado pela q88 (06/11/2026): o pagamento continua na reserva (RN-02.14).
- RN-06.6: o fundador não marcou se há taxa administrativa sobre a multa repassada nem se ela pode ser descontada da caução. Danos seguem RN-06.5 (franquia).
- Documentos do condutor (CNH, passaporte) são dados pessoais: armazenamento e retenção seguem PEN-48 (LGPD).
