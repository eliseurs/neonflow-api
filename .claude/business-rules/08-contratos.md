# RN-08 — Contratos e revisão jurídica

Os seis contratos do Neon Flow vêm do pacote V15 (minutas v0.4, 07/11/2026), que **ainda não está homologado**. Eles entram na plataforma para a revisão da advogada indicada pelo fundador. Nenhuma minuta é publicada ao cliente nem coleta aceite vinculante antes da aprovação jurídica e da liberação do admin. Decisão estrutural: [ADR-0015](../../docs/adr/ADR-0015-contratos-versionados-e-revisao-juridica.md).

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-08.1 | São **seis documentos**: 01 Termos Gerais de Uso e Intermediação, 02 Termo de Parceria do Prestador (módulos A–E), 03 Programa de Indicações (com o Módulo B, que não é um 7º documento), 04 Ordem de Locação sem motorista, 05 Ordem de Transfer e 06 Ordem de Passeio | implementada (08/11/2026) | `sql/migrations/V20_contracts.up.sql` (`contract_documents`) | PEN-71 |
| RN-08.2 | Cada mudança de texto é uma **versão nova**, com o hash SHA-256 do texto. Versões antigas, decisões e comentários **nunca são alterados nem apagados** (append-only, garantido no banco) | implementada (08/11/2026) | `contract.Hash`, triggers `contract_*_append_only` (V20) | — |
| RN-08.3 | Situação de cada versão: `minuta` → `enviado_para_revisao` → (`ajustes_solicitados`, e a correção vira uma versão nova) ou `aprovado_pela_revisora` → `liberacao_administrativa` → `publicado` | implementada (08/11/2026) | `internal/domain/contract/entity.go` (`transitions`, `Next`) | — |
| RN-08.4 | **Só a revisora jurídica** (`legal_reviewer`) pede ajustes, aprova ou revoga. O admin cria versões, envia para revisão, libera e publica, mas **não pode aprovar em nome da revisora**. A central não vê os contratos | implementada (08/11/2026) | `contract.Next`, `contract_route.go` (admin e legal_reviewer) | PEN-68 |
| RN-08.5 | A aprovação vale para a **versão exata**: a revisora envia o hash do texto que leu e uma confirmação expressa. O evento guarda hash, identidade autenticada (do JWT), papel e data e hora. Uma versão nova começa como minuta, sem herdar a aprovação | implementada (08/11/2026) | `contract.Next` (`ErrHashMismatch`), `EventRequest.Validate` (`ErrConfirmRequired`) | — |
| RN-08.6 | Pedir ajustes e revogar exigem justificativa. A revogação volta a versão para `enviado_para_revisao` e vale até a liberação. Depois de publicada, a correção é uma versão nova | implementada (08/11/2026; a revogação depois da liberação é decisão técnica) | `EventRequest.Validate`, `transitions` | — |
| RN-08.7 | **Aprovação jurídica não publica o contrato.** A publicação é uma decisão posterior do admin, depois das validações fiscal, regulatória, securitária e de PSP | implementada (08/11/2026) | `transitions` (`publicar` só a partir de `liberacao_administrativa`) | PEN-71 |
| RN-08.8 | A revisora comenta por cláusula (`clause`, ex.: 4.2), como observação ou pedido de correção, e pode propor uma redação substitutiva. O admin também comenta | implementada (08/11/2026) | `contract.CommentRequest`, `contract_comments` (V20) | — |
| RN-08.9 | O texto das minutas **não vai para o Git**: entra pela API (`POST /api/contracts/:code/versions`). Texto de contrato e comentários não vão para o log | implementada (08/11/2026; "Não subir este pacote ao Google Drive ou ao Git sem autorização do fundador") | `contract_handler.go` (`respond`) | — |

**Fontes:** pacote `09_CONTRATOS_V15_REVISAO_JURIDICA_E_DEV_2026-11-07` no Drive do fundador: `00_LEIA_PRIMEIRO.md`, `05_ESPECIFICACAO_DEV_V15.md`, `06_GUIA_AREA_JURIDICA_DRA_MONICA.md` e `04_CONTRATOS_V0_4_COPIAR_COLAR.txt` (lidos em 08/11/2026).

## Fora desta etapa

- Ordem 04 montada com os dados da reserva e aceite no padrão V15 (RN-02.3): depende de PEN-70.
- Convite por e-mail e MFA da revisora (PEN-68). Relatório em PDF e diferença entre versões (PEN-69).
- Situação do cadastro de veículo (`PENDENTE_TITULARIDADE`, `PENDENTE_AUTORIZACAO`, `PENDENTE_SEGURO`, `PENDENTE_VISTORIA`, `APROVADO`, `BLOQUEADO`) e a separação entre titular, cadastrante, locador contratual e beneficiário do repasse (05_ESPECIFICACAO_DEV): depende de PEN-70 e PEN-48.
