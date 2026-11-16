# Decisões: o que falta, o que está decidido e o que está feito

Leitura das 79 pendências de [pendencias.md](pendencias.md) por estado e severidade, conferida em 11/11/2026 (atualizada em 16/11/2026: login de parceiros e clientes, PEN-74 decidida, PEN-78 e PEN-79 novas; PEN-73 e PEN-77 resolvidas pelo pedido do site) contra o código da API, o app, as regras (`.claude/business-rules/`) e os ADRs. O registro continua sendo a fonte do status de cada PEN; este arquivo organiza o que fazer com elas. Ao resolver uma pendência, atualize os dois.

Severidade: **Bloqueante** impede operar com cliente real · **Alta** necessária para o primeiro ciclo persistente · **Média** precisa de decisão antes do MVP · **Baixa** higiene.

| Estado | Bloqueante | Alta | Média | Baixa | Total |
| --- | --- | --- | --- | --- | --- |
| [1. Falta decidir](#1-falta-decidir) | 4 | 9 | 6 | 0 | 19 |
| [3. Decidido ou sem decisão pendente, falta fazer](#3-falta-fazer) | 3 | 8 | 5 | 1 | 17 |
| [4. Decidido e feito](#4-decidido-e-feito) | 9 | 22 | 9 | 3 | 43 |

A seção [2](#2-revisões-de-classificação-propostas) lista mudanças de severidade e reaberturas propostas na conferência, **ainda não aplicadas** ao registro.

## 1. Falta decidir

Alguém precisa responder ou informar algo antes de haver o que implementar.

### Bloqueante

| PEN | O que falta decidir ou informar | Quem | O que isso trava |
| --- | --- | --- | --- |
| PEN-48 | Prazo de retenção de CNH e documentos (q75 e q103, com o advogado), base legal por finalidade e quem pode ver os documentos | Fundador + jurídico | Guardar documentos do cliente (RN-04.4), condutor adicional (RN-06.7), bloqueio por idade e tempo de habilitação (RN-06.8), portal do cliente (RN-04.6), documentos e Pix dos parceiros (RN-07.7) |
| PEN-71 | Razão social, CNPJ, endereço, suporte, política de privacidade, seguradora e apólice, política de cancelamento e percentuais das minutas 01 a 04 | Fundador + jurídico | Publicar qualquer contrato. Sem a Ordem 04 publicada, o aceite usa o texto de Termos, que não passou por advogado (pacote 5) |
| PEN-01 | Documentos e situação de cada veículo (CRLV, seguro, autorização do dono) e valor do km excedente | Fundador / operação | Ofertar a frota com segurança. O cadastro ainda não tem campo para documento nem situação do veículo |
| PEN-03 | Validade da habilitação estrangeira (advogado, q64), seguradora e valor da franquia (RN-06.5), taxa administrativa sobre multa e desconto da caução (RN-06.6) | Fundador + jurídico | Alugar sem motorista para estrangeiro; cobrar dano e multa com regra clara |

### Alta

| PEN | O que falta decidir ou informar | Quem | O que isso trava |
| --- | --- | --- | --- |
| PEN-72 | Para onde vai o aluguel com motorista na entrada do site: some, vira "passeio" ou fica dentro do aluguel | Fundador | Pacote 1; contrato 06 (passeio) |
| PEN-75 | "Assinar digitalmente" é o aceite com caixa e hash que já existe ou assinatura com provedor? | Fundador + jurídico | Pacote 5 |
| PEN-76 | A comissão do Neon Flow varia por serviço? O percentual do indicador incide sobre o valor do serviço ou sobre a comissão da plataforma? | Fundador | Pacote 6 |
| PEN-70 | Modelo contratual do lançamento: Neon Flow intermediador (parceiro é o locador) ou central operando a frota | Fundador + jurídico | Ordem 04 preenchida com os dados da reserva; situação do cadastro do veículo |
| PEN-67 | Transfer com carro de parceiro dirigido por outra pessoa: o dono recebe? Quanto e sobre qual base? | Fundador | Repasse do transfer (hoje só o motorista recebe) |
| PEN-65 | Acordos com hotéis e autorização de uso de nome e foto | Fundador | Comissão de hotel com acordo válido (RN-03.2); divulgação com parceiros |
| PEN-14 | Decidido em 16/11/2026 o que o parceiro vê (RN-07.8 a 07.11), o que o cliente faz (RN-02.33 a 02.37) e o aceite pelo motorista (PEN-74). Fica aberto só o que outros perfis venham a ter | Produto | — |
| PEN-11 | Metas dos indicadores | Fundador | Medir antes de expandir (`produto/metricas.md` com metas em branco) |
| PEN-37 | Namespace técnico `neoflow` × `neonflow` ([ADR-0005](adr/ADR-0005-namespace-tecnico.md), não decidido) | Fundador + arquitetura | Integração com o Nexus; nenhum código cria namespace até lá |

### Média

| PEN | O que falta decidir ou informar | Quem | O que isso trava |
| --- | --- | --- | --- |
| PEN-18 | Haverá checklist de vistoria na retirada, como o da devolução? (a q80 citou a vistoria da Localiza) | Produto + operação | Pacote 3 (vistoria) |
| PEN-19 | Como modelar proposta, negociação e pedido de ajuste | Produto | "Pedir ajuste" do pacote 2 |
| PEN-20 | Confirmar a ordem dos sete pacotes como nova sequência (o roadmap de 29/09 está defasado) | Fundador | Planejamento |
| PEN-41 | IA: provedor, fontes, ferramentas, avaliações e canal (API do WhatsApp Business) | Arquitetura + fundador | Pacote 7 |
| PEN-10 | A venda de página/bio (~R$ 400) entra no produto? | Fundador | Nada no curto prazo |
| PEN-39 | Fronteiras com NeoAuto, Neo360, NeoReferral e NeoPath | Arquitetura | Integrações entre verticais (RN-04.1) |

### Pontos em aberto sem PEN própria

Estão anotados nas regras, sem pendência aberta. Entram aqui para não se perderem.

| Ponto | Onde está anotado |
| --- | --- |
| Preço do aluguel no site: o ADR-0011 diz "visível", a RN-01.9 diz que a página do carro não mostra preço ("interpretação a confirmar") | [RN-01.9](../.claude/business-rules/01-frota.md), PEN-59 |
| Devolução atrasada que colide com a reserva seguinte do mesmo carro: o que a central faz | [RN-02.19](../.claude/business-rules/02-reservas.md), pontos em aberto |
| Valor da hora extra de espera no transfer | RN-02.16 |
| Percentual do Neon Flow sobre carro de parceiro (o exemplo do fundador é 30%, "em discussão") | [RN-03.7](../.claude/business-rules/03-financeiro-e-comissoes.md) |
| Manutenção inesperada: tratamento dos clientes afetados; não há bloqueio de manutenção por período | RN-01, detalhes |
| Base legal para manter registros financeiros depois da anonimização | [RN-04.8](../.claude/business-rules/04-dados-e-integracoes.md) |

## 2. Revisões de classificação propostas

Resultado da conferência de 11/11/2026. **Nada daqui foi aplicado**: mudar severidade ou reabrir uma decisão do fundador precisa de aprovação.

| PEN | Hoje | Proposta | Motivo |
| --- | --- | --- | --- |
| PEN-03 | Bloqueante, parcial | Alta | Quase tudo decidido; o site já manda o estrangeiro consultar a central |
| PEN-35 | Bloqueante, aberta | Média | O ADR-0004 diz que "bloqueia a implementação", mas ela foi autorizada e feita com autenticação própria (ADR-0009, PEN-52); os critérios do dia 1 (PEN-13) não citam o Nexus. Bloqueia só a integração |
| PEN-36 | Bloqueante, aberta | Média | Mesmo caso da PEN-35 |
| PEN-49 | Bloqueante, aberta | Alta, ligada à PEN-14 | Um só operador e nenhum tenant no código; os três papéis já são isolados por rota. Passa a bloquear quando parceiro, motorista ou cliente tiver login. Desde 16/11/2026 eles entram, ainda sem rota de dados: cada área nova tem de filtrar pelo cadastro da conta |
| PEN-42 | Alta, aberta | Baixa | Localizar um PDF histórico deixou de ser necessário para o ciclo persistente |
| PEN-59 | Alta, resolvida ("preço visível") | Reabrir | O site não mostra preço por decisão posterior (RN-01.9), e a API pública não devolve a diária. Duas decisões em conflito |
| PEN-50 | Alta, resolvida ("já consultei, e não precisa") | Voltar a parcial | Depois vieram os contratos para revisão de advogada, o modelo intermediador (PEN-70) e a habilitação estrangeira ainda por confirmar |

Se todas forem aceitas, as bloqueantes em aberto passam de 7 para 3: PEN-48, PEN-71 e PEN-01.

## 3. Falta fazer

Já decidido, ou não depende de decisão: é trabalho de engenharia, produto ou levantamento.

### Bloqueante

| PEN | O que falta fazer | Depende de |
| --- | --- | --- |
| PEN-35 | Localizar os donos existentes de identidade, agenda, documentos, pagamentos, eventos e permissões no Nexus | Acesso ao Nexus |
| PEN-36 | Auditar o Replit | Acesso ao Replit |
| PEN-49 | Isolamento por tenant e por perfil | PEN-14 (perfis), PEN-35 (modelo de tenant do Nexus) |

### Alta

| PEN | O que falta fazer | Depende de |
| --- | --- | --- |
| PEN-30 | Financeiro: cancelamento do aluguel cobrando 1 diária depois das 48h (RN-02.21), registrar a obrigação de estorno da reserva paga e cancelada (RN-03.3, RN-03.11) e contabilizar a caução à parte (RN-03.1) | Nada; RN-03.1 e RN-03.3 ainda estão como "proposta" |
| PEN-15 | Área do cliente (o login existe desde 16/11/2026, RN-05.13) e documentos no cadastro | PEN-48, PEN-14 |
| PEN-16 | Área do parceiro e documentos dos parceiros | PEN-14, PEN-48 |
| PEN-17 | Mostrar ao indicador o resultado das indicações | PEN-14 |
| PEN-68 | Convite individual e MFA da revisora jurídica; retirar a exceção que deixa o admin aprovar (RN-08.4) | Conta da advogada (o envio de e-mail existe desde 16/11/2026, ADR-0016) |
| PEN-38 | Especificar os contratos de integração com Shell e Nexus | PEN-35 |
| PEN-42 | Localizar `NEON_FLOW_cartao_executivo.pdf` | Fundador |
| PEN-78 | Verificar no Resend um domínio do Neon Flow para o remetente; sem ele nenhum e-mail chega a clientes e parceiros | Fundador (domínio e DNS) |

Sem PEN própria, decidido e ainda não implementado:

| O que falta fazer | Regra ou decisão | Depende de |
| --- | --- | --- |
| Formulário do site por serviço (retirada e devolução com hora; origem, destino, horário e passageiros) | Pacote 1 | Em parte, PEN-72 |
| Telas do pedido do site: pedir logado, acompanhar em Minha conta, lista da central com confirmar, recusar, pedir ajuste e avisar no WhatsApp (API pronta, RN-02.33 a 02.37) | Pacote 2 | — |
| Extensão do aluguel pela central, com cobrança da diferença | RN-02.20 | — |
| Pagamento online pela Stripe e pré-autorização da caução | ADR-0013, RN-06.4 | Depois do lançamento |
| CNH e documentos do cliente, condutor adicional, bloqueio por idade | RN-04.4, RN-06.7, RN-06.8 | PEN-48 |
| Ordens 05 (transfer) e 06 (passeio) e contratos 01 a 03 nos fluxos | RN-08, "fora desta etapa" | PEN-70, PEN-71 |
| Avaliação dos clientes nos indicadores | PEN-11 (q79) | Coletar avaliações |

### Média

| PEN | O que falta fazer | Depende de |
| --- | --- | --- |
| PEN-69 | Relatório de revisão em PDF e diferença entre versões dos contratos | — |
| PEN-79 | Limitar chamadas nas rotas públicas de cadastro e de nova senha | — |
| PEN-51 | QA visual, de teclado e de acessibilidade | — |
| PEN-12 | Validar mercado: concorrência, preço e demanda | Produto |
| PEN-45 | Ler os documentos citados e nunca abertos (arquitetura do Nexus, recuperação do projeto, snapshot do Replit) | Acesso aos arquivos |

Sem PEN própria: mostrar o dólar como conversão no site (ADR-0013, PEN-63), que depende de o site mostrar preço (PEN-59).

### Baixa

| PEN | O que falta fazer | Depende de |
| --- | --- | --- |
| PEN-34 | Servir as fontes localmente em vez do Google Fonts (`neonflow-app/index.html`) | — |

## 4. Decidido e feito

"Feito" indica se a decisão está implementada no sistema. Detalhes e datas ficam no registro.

### Bloqueante

| PEN | O que foi decidido | Feito |
| --- | --- | --- |
| PEN-02 | Tarifas configuráveis, mínimo de 1 diária, tanque cheio-cheio, km livre ou limitado, 1h de tolerância e atraso por minuto, extensão pela central | Sim, menos a extensão (RN-02.20) |
| PEN-04 | Pré-reserva bloqueia na criação e expira em 24h sem pagamento (ADR-0012). A PEN-73 pode rever isso para o pedido do site | Sim |
| PEN-05 | Aceite, pagamento integral na reserva por Pix ou link, caução na retirada, Stripe para depois (ADR-0013). Assinatura: PEN-75 | Sim, com pagamento confirmado pela central |
| PEN-06 | Cancelamento: transfer grátis até 24h, depois 50%; aluguel grátis até 48h, depois 1 diária | Transfer sim; aluguel não (PEN-30) |
| PEN-13 | Indispensáveis do dia 1: landing PT/EN, pedido por WhatsApp ou formulário, cadastros, reserva sem conflito, aceite, checklist com fotos e comissão de hotéis | Sim; o aceite exige um texto de Termos publicado |
| PEN-21 | Persistência no PostgreSQL | Sim |
| PEN-22 | Validação no servidor | Sim |
| PEN-23 | Autenticação e papéis (admin, central, revisora) | Sim |
| PEN-24 | Bloqueio atômico de período no banco | Sim |

### Alta

| PEN | O que foi decidido | Feito |
| --- | --- | --- |
| PEN-74 | Oferta aberta: o primeiro motorista que aceita leva; ele mesmo pode desistir; aviso pela central no WhatsApp (RN-02.30, RN-02.31, 16/11/2026) | Sim na API; telas no app a fazer |
| PEN-73 | Pedido do site não bloqueia; bloqueia quando a central confirma e a reserva nasce (RN-02.33, 16/11/2026) | Sim na API; telas no app a fazer |
| PEN-77 | Retorno por e-mail, Minha conta e botão de WhatsApp na central (RN-02.36, 16/11/2026) | E-mail e histórico na API; telas e botão no app a fazer |
| PEN-07 | Transfer: 60 min inclusos, espera extra cobrada, primeiro que aceita leva, oferta que só aumenta, desistência sem penalidade | Sim; aceite pelo motorista segue na PEN-74 |
| PEN-08 | Comissão por parceiro, sobre as diárias, saindo da parte do Neon Flow, gerada no encerramento | Sim; base e variação por serviço seguem na PEN-76 |
| PEN-09 | Neon Flow recebe, fica com percentual configurável e repassa ao dono; motorista recebe o valor da oferta | Sim; caso do carro de terceiro segue na PEN-67 |
| PEN-25 | Data inexistente recusada | Sim |
| PEN-26 | Etapas em ordem, sem avançar além do encerramento | Sim |
| PEN-27 | Valor gravado na reserva | Sim |
| PEN-28 | Frota mostra "Reservado" e a próxima reserva | Sim |
| PEN-29 | Data e hora livres, diária em blocos de 24h, fuso de Gramado (suposição técnica) | Sim |
| PEN-40 | Stack igual à dos projetos cognia e finder (ADR-0009) | Sim |
| PEN-43 | Números históricos não são regras nem metas | Encerrada como superada |
| PEN-50 | Fundador declarou que não precisa de parecer jurídico | Contestada: ver seção 2 |
| PEN-52 | Landing e reserva autorizadas (ADR-0011) | Sim |
| PEN-53 | Home com aluguel e transfer, com o mesmo peso | Sim |
| PEN-54 | Aluguel com e sem motorista. A PEN-72 pode rever a entrada do site | Sim |
| PEN-58 | Aeroportos e cidades: lista base e demais sob demanda | Sim |
| PEN-59 | Aluguel com preço visível, transfer sob consulta | Não: o site não mostra preço. Contestada: ver seção 2 |
| PEN-60 | Cliente fecha por WhatsApp, formulário e pagamento online | Em parte: o formulário só abre o WhatsApp; pagamento online depois |
| PEN-61 | Categorias Economy e Premium | Sim |
| PEN-66 | Número de WhatsApp da central | Sim |

### Média

| PEN | O que foi decidido | Feito |
| --- | --- | --- |
| PEN-31 | Testes do protótipo no CI do app | Sim |
| PEN-32 | Lint, testes e cobertura mínima de 80% em todo PR | Sim |
| PEN-44 | Nome Neon Flow, vertical única (ADR-0002) | Sim |
| PEN-55 | Shuttle compartilhado não entra | — |
| PEN-56 | Pacotes ficam para depois | — |
| PEN-57 | Convênio corporativo não entra | — |
| PEN-62 | Fotos mistas, reais e ilustrativas | Sim |
| PEN-63 | Português e inglês; cobrança em reais, dólar só como conversão | Idiomas sim; conversão não |
| PEN-64 | Logo do Neon Flow | Sim |

### Baixa

| PEN | O que foi decidido | Feito |
| --- | --- | --- |
| PEN-33 | Erros do README e data fixa do protótipo | Encerrada sem correção: protótipo congelado |
| PEN-46 | Retificação da regra de comissão registrada na RN-03.2 | Sim; protótipo congelado |
| PEN-47 | Decisões registradas em ADR | Sim |
