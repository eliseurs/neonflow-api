# Segurança e LGPD

O produto vai tratar dados pessoais (nome, contato) e documentos (CNH), além de dados de pagamento. Nada disso existe hoje, e o protótipo usa só dados fictícios.

## Requisitos mínimos antes do primeiro dado real

| Tema | Requisito | Pendência |
| --- | --- | --- |
| Base legal | Definir a base legal por finalidade (execução de contrato, legítimo interesse, consentimento) | PEN-48 |
| Minimização | Coletar só o necessário. Não replicar CNH nem caução entre verticais ([RN-04.1](../.claude/business-rules/04-dados-e-integracoes.md)) | PEN-48 |
| Retenção | Prazo de guarda de CNH, contrato e histórico | PEN-48 |
| Acesso | Autenticação e controle por perfil e por tenant | PEN-23, PEN-49 |
| Segredos | Nenhuma credencial no código. Usar variável de ambiente ou cofre. `.env` bloqueado no Claude Code | — |
| Logs | Nenhum dado pessoal em log | — |
| Pagamento | Não armazenar dados de cartão. Usar o provedor | PEN-05 |
| Jurídico | Contrato de locação, transfer e intermediação revisados | PEN-50 |

Revisões de segurança: `trinca-toolbox:security-reviewer` no diff e `/security-review` antes do PR.
