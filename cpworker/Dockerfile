# Estágio 1: Compilar o executável Go
FROM golang:1.24-alpine as builder

# Define o diretório de trabalho dentro do contêiner
WORKDIR /app

# Copia os arquivos de gerenciamento de dependências
COPY go.mod go.sum ./

# Baixa as dependências do projeto
RUN go mod download

# Copia todo o código-fonte do projeto para o contêiner
COPY . .

# Compila o binário do cpworker, otimizado para Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/cpworker ./cpworker

# Estágio 2: Criar a imagem final, leve e otimizada
FROM alpine:latest

# Copia apenas o binário compilado do estágio anterior
COPY --from=builder /go/bin/cpworker /go/bin/cpworker

# Define o comando que será executado quando o contêiner iniciar
ENTRYPOINT ["/go/bin/cpworker"]