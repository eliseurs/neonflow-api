# ADR-0013 — Pagamento integral online e preço em duas moedas

- **Status:** aceita
- **Data:** 04/11/2026
- **Decisores:** fundador
- **Relacionados:** PEN-02, PEN-05, PEN-63, ADR-0011

## Contexto e problema

O ADR-0011 definiu três canais para fechar a reserva (WhatsApp, formulário e pagamento online) e o site em português e inglês, mas não definiu o momento do pagamento, como o cliente aceita as condições sem assinatura eletrônica, nem a moeda.

## Decisão

| Tema | Decisão | Pergunta |
| --- | --- | --- |
| Momento do pagamento | **Valor integral, ao reservar.** Um pedido por WhatsApp ou formulário também é pago na reserva | q37 |
| Meio de pagamento no lançamento | **Pix ou link de pagamento, confirmado pela central.** O pagamento online integrado entra depois, sem segurar o lançamento | q53 |
| Contrato | **Aceite dos termos no site**, registrado. Sem assinatura eletrônica por enquanto | q38, q18 |
| Moeda | **Real e dólar.** O dólar é **só exibido**, como conversão na tela para o turista estrangeiro; a cobrança é sempre em reais | q30, q62, q63 |
| Valores | Tarifas, referências do transfer e hora extra de espera são **configuráveis** pela operação; nenhum valor fica no código | q26, q31 |

## Consequências

- O provedor de pagamento é uma escolha da engenharia (PEN-05). Ele precisa aceitar cartão internacional e pré-autorização para a caução (RN-06.4).
- O aceite precisa guardar a data e a hora, a versão dos termos aceita e a reserva. Isso exige versionar o texto dos termos.
- A cotação do dólar precisa de fonte e frequência de atualização. Como só é exibida, não entra na conciliação.
- No lançamento, a reserva precisa de uma ação da central para marcar o pagamento como confirmado (Pix ou link). A pré-reserva que a central não confirmar em 24h expira (RN-02.11).
- O conflito da q47 (pagamento online fora dos indispensáveis do dia 1) foi resolvido em 05/11/2026 pela q53.

## Fontes

- Questionário "Decisões do Neon Flow", q18, q26, q30, q31, q37, q38 e q47 (03 e 04/11/2026); q53, q62 e q63 (05/11/2026).
