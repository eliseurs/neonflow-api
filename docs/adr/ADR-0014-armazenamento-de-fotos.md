# ADR-0014 — Onde guardar as fotos dos veículos

- **Status:** proposta
- **Data:** 06/11/2026
- **Decisores:** time de engenharia, com aprovação do responsável pelo projeto
- **Relacionados:** RN-01.7, PEN-65, ADR-0008, ADR-0009

## Contexto e problema

O fundador decidiu que o cadastro do veículo aceita fotos e que o site usa imagens ilustrativas até elas existirem (RN-01.7, q85, 06/11/2026). Hoje a API só guarda dados no PostgreSQL e roda em um container sem disco persistente no Railway (ADR-0008): um arquivo gravado no container some a cada deploy. É preciso escolher onde as fotos ficam antes de implementar o upload.

## Fatores de decisão

- Fotos sobrevivem a deploys e reinícios, com backup.
- O site público (landing) e o painel mostram as fotos rápido, sem passar pela API a cada imagem.
- Custo baixo no volume atual (2 veículos, poucas fotos cada) e previsível se crescer.
- Pouca dependência nova e pouco segredo novo para gerenciar.
- As fotos são da frota, não de clientes: não são dados pessoais (LGPD fora do caminho, PEN-48).

## Opções consideradas

1. **Volume do Railway montado na API.** A API grava o arquivo em disco e serve por uma rota própria.
2. **Armazenamento de objetos compatível com S3 (Cloudflare R2 ou AWS S3).** O painel envia a foto direto para o bucket por uma URL assinada que a API gera; o banco guarda só a chave e a ordem; o site lê pela URL pública (CDN).
3. **Imagem no PostgreSQL (`bytea`).** A API grava e serve a imagem a partir do banco.

## Decisão

Recomendação, ainda não decidida: **opção 2, com Cloudflare R2.**

- O arquivo não passa pela API nem ocupa o banco; o site carrega a foto da CDN.
- R2 não cobra transferência de saída, e o volume atual cabe na faixa gratuita.
- Exige uma conta Cloudflare, um bucket e três variáveis de ambiente (`R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`) no Railway, e **uma dependência nova** na API para assinar as URLs (`aws-sdk-go-v2/service/s3` ou `minio-go`). Pela regra do projeto, a dependência nova precisa de aprovação.

## Consequências

- Boas: fotos duráveis e rápidas, API sem disco, custo próximo de zero no início, o mesmo caminho serve depois para documentos de clientes (com bucket privado e as regras de PEN-48).
- Ruins / riscos: um fornecedor a mais, segredos a guardar e girar; a opção 1 seria mais simples de montar, mas prende a API a uma única instância e complica o backup; a opção 3 incha o banco e os backups e deixa a landing dependente da API para cada imagem.

## Fontes

- Questionário "Decisões do Neon Flow", q85 (06/11/2026).
- `docs/deploy-railway.md`, ADR-0008 (hospedagem no Railway sem volume).
