package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	url         string
	total       int
	concurrency int
	timeout     time.Duration
	outputJson  bool
)

type result struct {
	Status   int
	Duration time.Duration
	Err      error
}

func main() {
	flag.StringVar(&url, "url", "", "URL do serviço *")
	flag.IntVar(&total, "requests", 1, "Total de requisições")
	flag.IntVar(&concurrency, "concurrency", 10, "Chamadas concorrentes")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "Timeout por requisição")
	flag.BoolVar(&outputJson, "json", false, "Imprimir saída em JSON")
	flag.Parse()

	fmt.Println("Programa de Benchmarking de Serviços HTTP")
	fmt.Println("-----------------------------")

	fmt.Printf("URL: %s\nTotal de requisições: %d\nConcorrência: %d\nTimeout: %s\nSaída em JSON: %v\n",
		url, total, concurrency, timeout, outputJson)

	fmt.Println("-----------------------------")

	if url == "" || total <= 0 || concurrency <= 0 {
		fmt.Println("Argumentos inválidos:")
		if url == "" {
			fmt.Println("- URL não pode ser vazia.")
		}
		if total <= 0 {
			fmt.Println("- Total de requisições deve ser maior que zero.")
		}
		if concurrency <= 0 {
			fmt.Println("- Concorrência deve ser maior que zero.")
		}
		return
	}

	if concurrency > total {
		concurrency = total
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency,
		},
	}

	jobs := make(chan int, total)
	results := make(chan result, total)

	var wg sync.WaitGroup
	var started int32

	worker := func() {
		defer wg.Done()
		for range jobs {
			start := time.Now()
			resp, err := client.Get(url)
			d := time.Since(start)

			if err != nil {
				results <- result{Status: 0, Duration: d, Err: err}
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			results <- result{Status: resp.StatusCode, Duration: d}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	for i := 0; i < total; i++ {
		atomic.AddInt32(&started, 1)
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	close(results)

	fmt.Println("Resultados:")

	for r := range results {
		fmt.Printf("Status: %d | Duração: %s | Erro: %v\n", r.Status, r.Duration, r.Err)
	}
}
