# go-stress-test

Ferramenta de benchmark e stress test para serviços HTTP.

## Como rodar

### 1. Go (sem Docker)

Execute na raiz do projeto:

```
go run main.go -url <URL> -requests <TOTAL> -concurrency <CONC> -timeout <TIMEOUT> [-json]
```

Ou compile e execute:

```
go build -o go-stress-test .
./go-stress-test -url <URL> -requests <TOTAL> -concurrency <CONC> -timeout <TIMEOUT> [-json]
```

### 2. Docker

Build da imagem:
```
docker build -t go-stress-test .
```

Execute o container:
```
docker run --rm go-stress-test -url <URL> -requests <TOTAL> -concurrency <CONC> -timeout <TIMEOUT> [-json]
```

## Parâmetros

- `-url` (obrigatório): URL do serviço a ser testado. Exemplo: `-url https://google.com`
- `-requests`: Total de requisições a serem feitas. Exemplo: `-requests 1000`
- `-concurrency`: Número de requisições concorrentes. Exemplo: `-concurrency 10`
- `-timeout`: Timeout por requisição (ex: `5s`, `1m`). Exemplo: `-timeout 10s`
- `-json`: (opcional) Imprime o relatório em formato JSON.

## Exemplo de uso

```
go run main.go -url https://google.com -requests 100 -concurrency 10 -timeout 5s
```

Ou com Docker:
```
docker run --rm go-stress-test -url https://google.com -requests 100 -concurrency 10 -timeout 5s
```