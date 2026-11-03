# Visão, identidade e fronteiras

## Nome do projeto

**Neon Flow**: vertical de aluguel e mobilidade do Nexus OS. O nome foi confirmado pelo fundador em 15/05/2026 e 27/09/2026 ([ADR-0002](../adr/ADR-0002-nome-neon-flow-vertical-unica.md)). Neonflow, NeonFlow, NeoFlow e Neo Flow são aliases de busca, não outras verticais.

## Domínio de negócio

- **Setor:** mobilidade e turismo.
- **Subdomínio:** aluguel de veículos, frota própria e de parceiros, reservas, transfer e motoristas parceiros.
- **Região:** Serra Gaúcha, com foco histórico em Gramado e Canela.
- **Público:** turistas e moradores, central de operação, condutores, motoristas, proprietários/locadoras e indicadores locais (hotéis, comércio).

## O negócio que orienta o produto

> "Neon Flow é a operação de aluguel, transfer, frota e experiência do turista na Serra Gaúcha, especialmente Gramado e Canela." (Revisão 29/09 §13)

Em 27/10/2025 o fundador descreveu o aluguel como o "ganha-pão" atual, com carros já disponíveis. Em 28/05/2026 corrigiu que o Neon Flow é uma empresa real de aluguel, transfer e turismo, não uma ferramenta de automação ou n8n.

## Missão

> TODO (bootstrap): preencher com o fundador. A wiki não inventa identidade.

## Visão

> TODO (bootstrap): preencher com o fundador.

## Tese de trabalho (proposta, não aprovada)

Organizar a mobilidade da viagem a partir de uma operação local confiável: aluguel como porta de entrada, transfer e passeios como continuidade, parceiros locais como distribuição e Nexus como memória contextual (Revisão 29/09 §14). É uma síntese, não um estudo de demanda (PEN-12).

## Fronteiras com o ecossistema

| Sistema | Responsabilidade | Relação com o Neon Flow |
| --- | --- | --- |
| **Neon Flow** | Oferta, frota, locação, transfer, atendimento e economia da vertical | Este projeto |
| Nexus Core | Identidade, contexto (CTX), eventos, governança e memória compartilhada | Fornece capacidades comuns ([ADR-0004](../adr/ADR-0004-reutilizar-capacidades-nexus.md)) |
| Shell | Interface contextual de cliente, central, motorista e parceiro | Entrada do cliente. Não é o painel administrativo |
| Orquestrador | Construção, dependências, custos tecnológicos, prontidão e evidências | Não deve fixar a comissão da vertical no código |
| NeoAuto | Compra, venda e liquidez de veículos | Vertical separada |
| Neo360 | Operação imobiliária | Conexão contextual futura, com autorização |
| NeoReferral | Rede de indicações | Mantém sua fronteira. Não fundir |
| NeoSerra | Contexto geográfico/ecossistêmico | Não é vertical criada por este projeto |

## Fora de escopo agora

NeoRent, NeoGuide, NeoStay, Wallet, tablets/kiosks e milhas são propostas históricas de expansão, não módulos autorizados (Dossiê B).

## Restrições

- Expansão subordinada à capacidade real. O fundador admitiu pausar se pesar sobre Neo360/Nexus (22/05/2026).
- Implementação exige autorização explícita (PEN-52).
