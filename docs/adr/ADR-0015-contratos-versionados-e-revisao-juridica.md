# ADR-0015 — Contratos em versões imutáveis com hash e revisão jurídica append-only

- **Status:** aceita
- **Data:** 08/11/2026
- **Decisores:** fundador (pacote V15), engenharia
- **Relacionados:** RN-08, RN-02.3, PEN-50, PEN-68, PEN-69, PEN-70, PEN-71

## Contexto e problema

O pacote V15 (07/11/2026) traz seis minutas contratuais v0.4 e pede uma área autenticada em que a advogada indicada pelo fundador revise cada cláusula e aprove a **versão exata**, com eventos imutáveis. O guia também diz que o admin não pode aprovar no lugar dela e que a aprovação não publica o contrato. O domínio `terms` (V17) guarda um único texto em versões, sem hash e sem fluxo de revisão.

## Fatores de decisão

- A aprovação precisa ficar presa ao texto lido: se o texto mudar, ela não vale para o novo.
- O histórico não pode ser apagado nem corrigido em silêncio, nem pelo admin nem por um bug.
- Tem de reaproveitar a autenticação existente (JWT + papéis), sem um login paralelo (06_GUIA).
- O `terms` atual não pode quebrar enquanto a Ordem 04 não o substituir.

## Opções consideradas

1. Estender `terms_versions` com tipo de documento e situação, guardada em coluna atualizada.
2. Domínio novo `contract`: versões com SHA-256, situação calculada a partir do último evento, e eventos e comentários append-only, com trigger no banco.
3. Provedor externo de assinatura ou revisão.

## Decisão

Opção 2. A situação nunca é gravada numa coluna: é o último evento. Cada transição trava a versão (`FOR UPDATE`) e confere, na mesma transação, o papel, a situação atual e, na aprovação, o hash enviado pela revisora. Triggers recusam `UPDATE` e `DELETE` em versões, eventos e comentários. O papel `legal_reviewer` entra em `users.role` e só acessa as rotas de contratos. O `terms` continua como está até a Ordem 04 (PEN-70).

## Consequências

- Boas: a aprovação pode ser auditada pelo hash. O admin não consegue se passar pela revisora, porque o papel vem do JWT e a regra fica no servidor. Corrigir um texto obriga a criar uma versão nova e passar por nova revisão.
- Ruins / riscos: um erro de digitação também vira versão nova. Um admin poderia criar para si um usuário `legal_reviewer`, e isso é controle organizacional (PEN-68). Por enquanto, os textos de contrato convivem com os termos de `terms`.

## Fontes

- `06_GUIA_AREA_JURIDICA_DRA_MONICA.md` (fluxo, estados e critérios de aceite 1 a 10), `05_ESPECIFICACAO_DEV_V15.md`, `00_LEIA_PRIMEIRO.md`; pacote V15, 07/11/2026.
