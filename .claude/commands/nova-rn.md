---
description: Registra uma regra de negócio (RN-NN.x) em .claude/business-rules a partir de uma fonte verificável
argument-hint: "<descrição da regra>"
---

Registre a regra de negócio: $ARGUMENTS

1. **Pergunte a fonte** se ela não estiver clara: fala do fundador, documento ou código. Regra sem fonte não entra.
2. Escolha o arquivo do tema em `.claude/business-rules/` (01-frota, 02-reservas, 03-financeiro-e-comissoes, 04-dados-e-integracoes, 05-autorizacao). Tema novo → novo arquivo `NN-tema.md` e linha no índice do `README.md`.
3. Adicione a linha na tabela com o próximo `RN-NN.x`, status e "Onde é aplicada":
   - `aprovada (DD/MM/AAAA)`: só com decisão explícita registrada.
   - `implementada`: existe na API com teste (`arquivo:linha`).
   - `protótipo`: existe só em `../neonflow-app/prototipo/dist/`.
   - `proposta`: sugestão sem aprovação.
4. Liste as pendências (`PEN-NN`) e atualize `docs/pendencias.md` se a regra resolver alguma.
5. Registre em `.claude/specs.md`.
