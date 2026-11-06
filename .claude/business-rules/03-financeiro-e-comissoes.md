# RN-03 — Financeiro e Comissões

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-03.1 | Caução é contabilizada separadamente da receita de locação ou transfer | proposta | — (o protótipo não contabiliza caução) | PEN-03, PEN-30 |
| RN-03.2 | Identificar a origem de uma solicitação não cria direito a comissão. Comissão exige acordo válido, base de cálculo, elegibilidade, conclusão, pagamento e tratamento de estorno | implementada em parte (06/11/2026): a comissão só nasce no encerramento, com reserva paga e finalizada (RN-03.12); acordo e elegibilidade por hotel seguem em aberto | `booking_repository.go` (`queryCreatePayouts`) | PEN-08, PEN-46 |
| RN-03.3 | Cancelar uma reserva paga preserva o recebimento e registra a obrigação de estorno | proposta | — (o protótipo remove a reserva do financeiro) | PEN-06, PEN-30 |
| RN-03.4 | O percentual de comissão é **configurável** pela operação. Nenhum percentual fica fixo no código. Em 06/11/2026 o fundador definiu que o percentual é **por parceiro** (antes: por categoria do que foi indicado) | aprovada (03/11/2026; por parceiro em 06/11/2026) | `partners.commission_bp` (RN-07.6); cálculo em RN-03.12 | PEN-08 |
| RN-03.5 | Manutenção e combustível do veículo parceiro são custos do **dono do veículo** | aprovada (03/11/2026, fundador) | — | PEN-09 |
| RN-03.6 | O pagamento do cliente é repassado ao dono do veículo **ao final do aluguel** | aprovada (03/11/2026, fundador) | — | PEN-09 |
| RN-03.7 | **O Neon Flow recebe o pagamento do cliente**, fica com um **percentual configurável** e repassa o restante ao dono do veículo. O exemplo do fundador é 30%, ainda em discussão | aprovada (04/11/2026, fundador; quem recebe primeiro confirmado em 05/11/2026, q51); percentual não definido | — | PEN-09 |
| RN-03.8 | A comissão do indicador e o pagamento do motorista são repassados **depois do pagamento concluído e do serviço finalizado** | aprovada (05/11/2026, fundador, q52) | — | PEN-08, PEN-09 |
| RN-03.9 | O motorista recebe um **percentual da corrida** | aprovada (04/11/2026, fundador); no transfer, aplicado como valor da oferta (RN-03.14, 06/11/2026) | `partners.driver_share_bp` (referência para a central definir a oferta) | PEN-09 |
| RN-03.10 | Dono e motorista são a **mesma pessoa só no transfer e no passeio** (aluguel com motorista): o dono dirige o próprio carro. No aluguel sem motorista, quem dirige é o cliente que alugou | aprovada (05/11/2026, fundador, q60) | — | PEN-09 |
| RN-03.11 | O **estorno** de um cancelamento é feito **manualmente pela central**. O valor segue RN-02.10 (transfer) e RN-02.21 (aluguel) | aprovada (06/11/2026, fundador, q72) | — | PEN-05, PEN-30 |
| RN-03.12 | **Comissão do indicador:** gerada quando a reserva indicada é **encerrada** (paga e com o serviço finalizado), com o percentual do parceiro **sobre as diárias** (sem atraso, km nem caução). A comissão **sai da participação do Neon Flow**: o repasse ao dono não muda com ou sem indicação. A central paga por Pix fora do sistema e **marca como paga** (data e quem marcou) | implementada (06/11/2026; fundador no CLI: base "Só as diárias", "Marcar como paga", "Da parte do Neon Flow") | `V11_partner_payouts.up.sql`, `booking_repository.go` (`queryCreatePayouts` no encerramento), `internal/domain/payout` | PEN-08 |
| RN-03.13 | **Repasse ao proprietário:** carro de parceiro (`owner_partner_id`) gera no encerramento o repasse ao dono, com o percentual do parceiro **sobre diárias + atraso + km excedente** (a caução nunca entra). O Neon Flow fica com o restante como **participação administrativa**. Carro próprio não gera repasse | implementada (06/11/2026; fundador no CLI: "quando o veiculo não é propriedade da plataforma tera apenas uma participação adminitrativa nos ganhos sobre um veiculo particular"; base "Diárias + atraso + km") | mesmos de RN-03.12 | PEN-09 |
| RN-03.14 | **Transfer:** o motorista recebe o **valor da oferta que aceitou** mais a **espera extra** inteira; a oferta é definida pela central e pode aumentar quando ninguém aceita (concilia RN-03.9 com RN-02.22). Indicação gera **comissão sobre o preço da corrida**, sem a espera, saindo da parte do Neon Flow. Os dois nascem quando a corrida paga é **concluída** e aparecem em Repasses | implementada (06/11/2026; fundador no CLI: "Valor da oferta", espera "Para o motorista", comissão "Sim, sobre o preço") | `V13_transfers.up.sql`, `transfer_repository.go` (`queryPayouts`), `internal/domain/payout` (tipo `motorista`) | PEN-09, PEN-08 |

**Fontes:** Revisão 29/09 §13–15; Dossiê C.5, D e F; questionário do fundador em 03/11/2026 (RN-03.4 a RN-03.6), 04/11/2026 (RN-03.7 a RN-03.9) 05/11/2026 (RN-03.7, RN-03.8, RN-03.10) e 06/11/2026 (RN-03.11; RN-03.12 a RN-03.14 no CLI).

## Detalhes

- "Comissão após pagamento e conclusão" foi formulada pela IA em 09/10/2025. A fala humana localizada dizia que os valores "não estavam estipulados" (PEN-46).
- Percentuais em conflito: ~15% por corrida indicada (fala do fundador, 27/08/2026) contra a matriz 20% app / 15% motorista (memória não reconfirmada) (PEN-08). **Não codificar nenhum dos dois.**
- O Orquestrador não deve fixar a comissão no código (Revisão 29/09 §15). RN-03.4 confirma: configurável por categoria. Ainda faltam base de cálculo, elegibilidade e estorno (PEN-08). "Categoria" foi entendida como categoria do serviço ou veículo indicado; confirmar.
- O conflito q43 × q44 sobre quem recebe primeiro foi resolvido em 05/11/2026 (q51): o Neon Flow recebe e repassa ao dono.
- RN-03.10: no transfer e no passeio, motorista e dono são a mesma pessoa, então a corrida se divide entre o Neon Flow e o dono-motorista (RN-03.7, RN-03.9). Falta dizer se RN-03.7 e RN-03.9 são o mesmo percentual ou dois percentuais diferentes nesse caso.
- RN-03.8: como a comissão só é paga com o serviço finalizado, uma reserva cancelada não gera comissão. Isso é consequência da regra, não uma resposta explícita do fundador.
- RN-03.5 vale para o veículo parceiro. No aluguel sem motorista, quem paga o combustível usado pelo cliente não foi definido (PEN-02).
- Números históricos (R$400 por bio, R$1.500/dia, SLA de 60s) não são regras nem metas (PEN-43).
