package main

import (
	"flag"
	"fmt"
	"time"
)

var (
	url         string
	total       int
	concurrency int
	timeout     time.Duration
	outputJson  bool
)

func main() {
	flag.StringVar(&url, "url", "", "URL do serviço *")
	flag.IntVar(&total, "requests", 1, "Total de requisições")
	flag.IntVar(&concurrency, "concurrency", 10, "Chamadas concorrentes")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "Timeout por requisição")
	flag.BoolVar(&outputJson, "json", false, "Imprimir saída em JSON")
	flag.Parse()

	fmt.Printf("URL: %s\nTotal de requisições: %d\nConcorrência: %d\nTimeout: %s\nSaída em JSON: %v\n",
		url, total, concurrency, timeout, outputJson)
}
