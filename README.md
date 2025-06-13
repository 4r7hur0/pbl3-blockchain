# PBL-2

## Descrição do Projeto

Este projeto simula um sistema distribuído de recarga de veículos elétricos, utilizando múltiplos serviços (empresas), um registry para descoberta de serviços e comunicação via MQTT (Mosquitto). O ambiente é orquestrado com Docker e Docker Compose, utilizando uma rede overlay para integração dos containers.

## Pré-requisitos
- Go 1.24 ou superior
- Docker e Docker Compose

## 1. Criando a rede overlay
A rede overlay permite que múltiplos containers, possivelmente em diferentes hosts (ou no mesmo host), se comuniquem de forma segura e isolada. É fundamental para a integração dos serviços deste projeto.

### O que é uma rede overlay?
Uma rede overlay do Docker conecta containers em diferentes máquinas (ou no mesmo host) como se estivessem na mesma rede local. É ideal para sistemas distribuídos e microserviços.

### Como criar a rede overlay
Execute o comando abaixo **apenas uma vez** (a rede persiste até ser removida):

```bash
docker network create --driver overlay --attachable rede-overlay
```

- `--driver overlay`: define o tipo da rede.
- `--attachable`: permite que containers standalone (não orquestrados por Swarm) se conectem à rede.
- `rede-overlay`: nome da rede (pode ser alterado, mas mantenha igual em todos os comandos do projeto).

### Como verificar se a rede já existe
```bash
docker network ls
```
Procure por `rede-overlay` na lista.

### Como remover a rede overlay (caso precise recriar)
```bash
docker network rm rede-overlay
```

### Dicas
- Sempre use o mesmo nome de rede nos comandos de execução dos containers.
- Se estiver usando Docker Swarm, inicialize com `docker swarm init` antes de criar a rede overlay.
- Para ambientes locais, o parâmetro `--attachable` é suficiente.

## 2. Rodando o Mosquitto (Broker MQTT)
O Mosquitto é necessário para a comunicação MQTT entre os serviços. Execute:

```bash
docker run -d --name mosquitto --network rede-overlay -p 1883:1883 -v /home/tec502/PBL-2/mosquitto/mosquitto.conf:/mosquitto/config/mosquitto.conf eclipse-mosquitto
```

> **Atenção:** Verifique se o caminho do arquivo `mosquitto.conf` está correto.

## 3. Rodando o Registry
O registry é responsável por registrar e localizar as APIs das empresas.

### Build do container Registry:
```bash
docker build -f registry/Dockerfile -t registry .
```

### Executando o Registry:
```bash
docker run -d --name registry --network rede-overlay -p 9000:9000 registry
```

## 4. Rodando as APIs das Empresas
Cada empresa deve ser executada em um terminal/container diferente, sempre na mesma rede overlay.

### Build do container da API:
```bash
docker build -f api/Dockerfile -t api .
```

### Executando uma empresa (exemplo):
```bash
docker run -d \
  --name SolAtlantico \
  --network rede-overlay \
  -p 8080:8080 \
  -e ENTERPRISE_NAME=SolAtlantico \
  -e ENTERPRISE_PORT=8080 \
  -e OWNED_CITY=Salvador \
  -e POSTS_QUANTITY=2 \
  -e REGISTRY_URL="http://registry:9000" \
  api
```

Repita o comando acima para cada empresa, alterando os valores das variáveis de ambiente e o nome do container. Exemplos:

```bash
docker run -d --name SertaoCarga --network rede-overlay -p 8081:8081 \
  -e ENTERPRISE_NAME=SertaoCarga \
  -e ENTERPRISE_PORT=8081 \
  -e OWNED_CITY="Feira de Santana" \
  -e POSTS_QUANTITY=5 \
  -e REGISTRY_URL="http://registry:9000" \
  api

docker run -d --name CacauPower --network rede-overlay -p 8083:8083 \
  -e ENTERPRISE_NAME=CacauPower \
  -e ENTERPRISE_PORT=8083 \
  -e OWNED_CITY=Ilheus \
  -e POSTS_QUANTITY=2 \
  -e REGISTRY_URL="http://registry:9000" \
  api
```

## 5. Executando os carros e o listEnterprises
Para rodar os carros e o serviço de listagem de empresas, utilize o Docker Compose:

```bash
docker compose up --build
```

Conecte o container do Mosquitto à rede bridge:
```bash
docker network connect nome-da-rede-bridge mosquitto
```

## Observações Importantes
- Sempre utilize a mesma rede overlay para todos os containers.
- O registry deve estar rodando antes das empresas.
- Cada empresa deve ser instanciada em um container separado.
- Verifique as portas para evitar conflitos.
- O Mosquitto deve estar acessível para todos os serviços.

---

Em caso de dúvidas, consulte os arquivos do projeto ou abra uma issue.