# ADR-0008 — Hospedar a demonstração no Railway com servidor Node sem dependências

- **Status:** aceita
- **Decisores:** engenharia
- **Relacionados:** ADR-0003, PEN-52, PEN-49

## Contexto e problema

O protótipo (`dist/`) roda hoje no Sites, com acesso privado ao proprietário (Dossiê I). Em 03/11/2026 o time pediu para hospedá-lo também no Railway a partir deste repositório. O Railway executa um processo que precisa escutar em `$PORT` e não serve diretório estático sozinho.

## Opções consideradas

1. **Servidor Node sem dependências** (`server.mjs`), detectado pelo Railpack via `package.json`.
2. Dockerfile com nginx ou Caddy.
3. Pacote npm `serve`.

## Decisão

Opção 1. O `server.mjs` serve só `dist/`, sem modificá-lo ([ADR-0003](ADR-0003-prototipo-estatico-e-evidencia.md)), escuta em `$PORT`/`0.0.0.0`, expõe `/healthz`, envia headers de segurança, bloqueia arquivos ocultos e travessia de diretório, e aceita Basic Auth opcional por variáveis de ambiente. A configuração fica em `railway.json`.

## Consequências

- Boas: nenhuma dependência para manter ou auditar. Seis testes automatizados cobrem o servidor.
- Ruins: Basic Auth é uma proteção mínima. Não substitui autenticação por perfil (PEN-23, PEN-49).
- **Sem `BASIC_AUTH_*` definidos, a URL do Railway fica pública**, ao contrário do Sites. A Base v1.1 e a revisão de 29/09 ficariam abertas a quem tiver o link. Recomendado: definir as variáveis.
- Hospedar a demonstração não autoriza implementar funcionalidades de produto (PEN-52 segue aberta).
