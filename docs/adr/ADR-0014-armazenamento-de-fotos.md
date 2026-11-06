# ADR-0014 — Onde guardar as fotos dos veículos

- **Status:** aceita (06/11/2026)
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

**Opção 2, armazenamento compatível com S3**, aprovada pelo responsável em 06/11/2026: "pode por hora guardar arquivos em um bucket no railway depois faço migração para cloudflare r2".

- **Agora:** bucket privado do Railway (`neonflow-arquivos`, região US East, ambiente hml), criado pela CLI.
- **Depois:** Cloudflare R2. A troca é só de variáveis, porque o código fala S3 (`STORAGE_ENDPOINT`, `STORAGE_REGION`, `STORAGE_BUCKET`, `STORAGE_ACCESS_KEY_ID`, `STORAGE_SECRET_ACCESS_KEY`), mais copiar os objetos para o novo bucket.
- **Dependência aprovada:** `github.com/minio/minio-go/v7` (Apache 2.0), só o cliente S3.
- **Diferença da recomendação inicial:** o envio passa pela API (multipart), que confere tipo pelo conteúdo e tamanho (10 MB) antes de gravar; a leitura é por link assinado de 15 minutos. O bucket é privado, inclusive para as fotos da reserva, que mostram placas e o estado do carro de cada cliente.
- **Sem variáveis:** a API sobe normalmente e o envio responde 503 "armazenamento de arquivos não configurado" (ambiente local).

## Consequências

- Boas: fotos duráveis e rápidas, API sem disco, custo próximo de zero no início, o mesmo caminho serve depois para documentos de clientes (com bucket privado e as regras de PEN-48).
- Ruins / riscos: um fornecedor a mais, segredos a guardar e girar; a opção 1 seria mais simples de montar, mas prende a API a uma única instância e complica o backup; a opção 3 incha o banco e os backups e deixa a landing dependente da API para cada imagem.

## Fontes

- Questionário "Decisões do Neon Flow", q85 (06/11/2026).
- `docs/deploy-railway.md`, ADR-0008 (hospedagem no Railway sem volume).
