# pbl3-blockchain

Este projeto implementa uma arquitetura blockchain utilizando Hyperledger Fabric, com chaincode desenvolvido em Go. O objetivo é fornecer uma infraestrutura robusta para aplicações descentralizadas, facilitando a automação de processos e a segurança de transações em ambientes permissionados.

## Índice

- [Requisitos](#requisitos)
- [Instalação do Hyperledger Fabric](#instalação-do-hyperledger-fabric)
- [Inicialização da Rede Hyperledger (teste-network)](#inicialização-da-rede-hyperledger-teste-network)
- [Deploy do Chaincode com deployCC](#deploy-do-chaincode-com-deploycc)
- [Executando a Blockchain com Docker Compose](#executando-a-blockchain-com-docker-compose)
- [Executando as APIs e Serviços com Docker Compose](#executando-as-apis-e-serviços-com-docker-compose)
- [Atenção aos Diretórios](#atenção-aos-diretórios)
- [Referências](#referências)

---

## Requisitos

Antes de começar, certifique-se de ter os seguintes itens instalados em seu ambiente:

- [Go (Golang)](https://go.dev/dl/) (versão recomendada: 1.20 ou superior)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Git](https://git-scm.com/)
- `curl` (opcional, para baixar dependências)
- Hyperledger Fabric Binaries e Docker Images

---

## Instalação do Hyperledger Fabric

Para rodar este projeto, é necessário baixar as ferramentas e imagens do Hyperledger Fabric. Execute os comandos abaixo em um terminal:

```bash
# Clone o repositório do fabric-samples (ele contém scripts úteis)
git clone https://github.com/hyperledger/fabric-samples.git

# Entre na pasta fabric-samples e rode o script bootstrap (ajuste a versão conforme necessário)
cd fabric-samples
curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.2.0 1.5.2
```

> **Nota:** O script `bootstrap` fará o download dos binários (`peer`, `orderer`, `configtxgen`, etc.) e das imagens Docker necessárias. Certifique-se de ter permissão para executar scripts shell.

---

## Inicialização da Rede Hyperledger (teste-network)

O Hyperledger Fabric inclui uma rede de teste chamada `test-network` que facilita a criação de uma infraestrutura blockchain local para desenvolvimento e testes.

### Passos para inicializar a rede:

1. **Acesse o diretório da test-network:**

   ```bash
   cd fabric-samples/test-network
   ```

2. **Suba a rede, crie um canal e utilize o Certificate Authority (CA):**

   ```bash
   ./network.sh up createChannel -ca
   ```

   - Este comando irá:
     - Inicializar os containers necessários (orderers, peers, CA, CouchDB, etc).
     - Criar um canal padrão chamado `mychannel` (pode ser alterado com o parâmetro `-c`).
     - Utilizar o CA para emissão dos certificados dos participantes.

> **Atenção:** Execute o comando acima sempre a partir do diretório `test-network`. Certifique-se de que os diretórios de artefatos e volumes estejam corretamente criados.

---

## Deploy do Chaincode utilizando o deployCC

Após a rede estar rodando, faça o deploy do chaincode (smart contract) utilizando o comando abaixo:

```bash
./network.sh deployCC -ccn pbl3 -ccp caminho/para/pbl3-blockchain/chaincode/ -ccl go
```
- `-ccn` define o nome do chaincode (aqui usamos `pbl3`).
- `-ccp` deve apontar para o caminho do diretório do seu chaincode (por exemplo: `/home/usuario/repos/pbl3-blockchain/chaincode/`).
- `-ccl` define a linguagem utilizada (neste projeto, `go`).

> **Importante:**  
> - O caminho passado em `-ccp` deve ser absoluto ou relativo ao diretório onde o comando está sendo executado (`test-network`).  
> - Se o diretório estiver incorreto, o deploy irá falhar.  
> - Sempre verifique se está no diretório correto antes de executar os scripts!

---

## Executando a Blockchain com Docker Compose

A test-network já utiliza internamente um arquivo `docker-compose` para subir todos os containers necessários do Hyperledger Fabric (orderers, peers, CA, CouchDB, etc).  
Se precisar subir ou verificar os containers manualmente, utilize:

```bash
docker ps -a
```

Para verificar os logs dos serviços:

```bash
docker-compose logs -f
```

> **Observação:** Os containers de blockchain devem estar rodando antes de subir suas APIs ou outros serviços que dependam da rede Fabric.

---

## Executando as APIs e Serviços com Docker Compose

Além dos containers da blockchain, este projeto inclui containers adicionais (por exemplo, APIs, frontends, serviços auxiliares) definidos em um ou mais arquivos `docker-compose.yml` próprios do projeto.

### Como executar suas APIs e serviços:

1. **Verifique o arquivo `docker-compose.yml` do seu projeto:**
   - Ele normalmente está na raiz do repositório ou em uma pasta específica para infraestrutura/deployment.

2. **Atenção aos diretórios mapeados:**
   - Certifique-se de que os volumes e caminhos utilizados no `docker-compose.yml` apontam para os diretórios corretos do seu projeto.  
   - Por exemplo, diretórios de código, arquivos de configuração, chaves ou certificados.  
   - Se rodar o comando em um subdiretório, os caminhos relativos podem não funcionar.

3. **Suba os containers:**
   ```bash
   docker-compose up -d
   ```
   ou, caso o arquivo esteja em outro diretório:
   ```bash
   docker-compose -f caminho/para/docker-compose.yml up -d
   ```

4. **Verifique os logs para garantir que as APIs subiram corretamente:**
   ```bash
   docker-compose logs -f
   ```

> **Atenção:**  
> - Sempre execute o `docker-compose` a partir do diretório correto para evitar problemas com caminhos relativos de volumes.  
> - Se suas APIs dependem da blockchain, garanta que a test-network já está rodando antes.  
> - Se alterar a localização dos diretórios, ajuste os volumes no `docker-compose.yml` correspondente.

---

## Atenção aos Diretórios

- **Mantenha a estrutura de diretórios padrão** tanto da Hyperledger Fabric quanto do seu projeto para evitar erros nos scripts e deploy.
- **Execute os scripts SEMPRE a partir do diretório correto** (`test-network` para comandos de rede e deploy da Fabric, raiz do projeto para APIs).
- **Verifique os caminhos absolutos e relativos** nos arquivos de configuração (`docker-compose.yml`, scripts, configs), principalmente se estiver rodando comandos a partir de subdiretórios.
- **Em caso de erros, confira se os diretórios e caminhos informados existem e têm permissão de acesso**.

---

## Referências

- [Documentação do Hyperledger Fabric](https://hyperledger-fabric.readthedocs.io/)
- [Fabric Samples no GitHub](https://github.com/hyperledger/fabric-samples)
- [Guia oficial do Docker Compose](https://docs.docker.com/compose/)

---

Caso tenha dúvidas ou encontre problemas, abra uma issue neste repositório!