# ADR-0016 — E-mails da API pelo Resend

- **Status:** aceita (16/11/2026)
- **Data:** 16/11/2026
- **Decisores:** fundador (pela conversa no CLI) e time de engenharia
- **Relacionados:** RN-05.11 a RN-05.14, RN-04.6, PEN-14, PEN-15, ADR-0004

## Contexto e problema

Em 16/11/2026 o fundador decidiu que parceiros (de qualquer tipo) e clientes entram na aplicação, que o cliente se cadastra sozinho e que a senha esquecida volta por e-mail. Para o cliente ver o histórico feito pela central sem que outra pessoa se cadastre com o e-mail dele, o e-mail precisa ser confirmado. A API não enviava e-mail nenhum.

## Fatores de decisão

- Confirmar o dono do e-mail antes de ligar a conta ao cadastro do cliente (LGPD, RN-04.8).
- Pouca dependência nova e um só segredo.
- O envio não pode derrubar a API: sem configuração, só as rotas que mandam e-mail param.

## Opções consideradas

1. **Resend** pela biblioteca oficial `resend-go`.
2. SMTP de um provedor qualquer, com a biblioteca padrão do Go.
3. Links mandados pela central no WhatsApp, sem e-mail.

## Decisão

**Opção 1**, pedida pelo responsável em 16/11/2026: "utilize o resend-go direto no railway eu vou setar a API-KEY".

- **Dependência aprovada:** `github.com/resend/resend-go/v3` v3.17.0 (MIT), que exige Go 1.23 e não traz outras dependências; o `go.mod` continua em Go 1.25.
- **Variáveis:** `RESEND_API_KEY` (segredo, no Railway), `EMAIL_FROM` (remetente) e `APP_URL` (endereço do neonflow-app, para os links). Sem a chave ou o remetente, cadastro, convite e nova senha respondem 503 "envio de e-mail não configurado" e o resto da API funciona.
- **Texto simples**, em português, com o link para a página do app (`/conta/confirmar/:token`, `/conta/convite/:token`, `/conta/nova-senha/:token`). O código do link tem 32 bytes aleatórios; o banco guarda só o hash (como os links de termos).
- **Log:** a falha do provedor vai para o log com o assunto, nunca com o destinatário.
- O Nexus não tem hoje um serviço de mensagens próprio para reaproveitar (ADR-0004, PEN-35); se tiver, a troca fica atrás da interface `email.Sender`.

## Consequências

- Boas: confirmação de e-mail e senha esquecida sem trabalho manual da central; um segredo só; troca de provedor isolada no pacote `internal/infra/email`.
- Ruins / riscos:
  - **O Resend só envia de domínio verificado.** Um remetente `@gmail.com` não pode ser verificado: sem domínio próprio, a conta só manda de `onboarding@resend.dev` e só para o e-mail do dono da conta, o que serve para teste, não para clientes. É preciso verificar no DNS um domínio do Neon Flow (PEN-78).
  - As rotas públicas de cadastro e de nova senha não têm limite de chamadas: alguém pode disparar e-mails para endereços de terceiros (PEN-79).

## Fontes

- Conversa no CLI com o responsável, 16/11/2026 (respostas sobre login de parceiros e clientes).
- Documentação do Resend sobre domínios verificados.
