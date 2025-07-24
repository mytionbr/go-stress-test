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

type report struct {
	URL            string
	TotalRequests  int
	Concurrency    int
	TotalTime      time.Duration
	RequestsPerSec float64
	HTTP200        int
	StatusDist     map[int]int
	Errors         int
	StartTime      time.Time
	EndTime        time.Time
	LatencySamples []time.Duration
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

	startTime := time.Now()

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

	rep := buildReport(url, total, concurrency, startTime, results)

	fmt.Println("--- Relatório de Teste de Carga ---")
	fmt.Printf("URL:            %s\n", rep.URL)
	fmt.Printf("Total:          %d\n", rep.TotalRequests)
	fmt.Printf("Concorrência:   %d\n", rep.Concurrency)
	fmt.Printf("Tempo Total:    %s\n", rep.TotalTime)
	fmt.Printf("RPS:            %.2f\n", rep.RequestsPerSec)
	fmt.Printf("HTTP 200:       %d\n", rep.HTTP200)
	fmt.Printf("Erros:          %d\n", rep.Errors)
	fmt.Println("Status HTTP:")
	for k, v := range rep.StatusDist {
		fmt.Printf("  %d: %d\n", k, v)
	}
	fmt.Printf("Início: %s\nFim:    %s\n", rep.StartTime.Format(time.RFC3339), rep.EndTime.Format(time.RFC3339))

}

func buildReport(url string, total, concurrency int, startTime time.Time, results <-chan result) report {
	statusDist := make(map[int]int)
	var http200, errs int
	var durations []time.Duration

	for r := range results {
		if r.Err != nil {
			errs++
			continue
		}
		statusDist[r.Status]++
		if r.Status == 200 {
			http200++
		}
		durations = append(durations, r.Duration)
	}

	endTime := time.Now()
	totalTime := endTime.Sub(startTime)

	return report{
		URL:            url,
		TotalRequests:  total,
		Concurrency:    concurrency,
		TotalTime:      totalTime,
		RequestsPerSec: float64(total) / totalTime.Seconds(),
		HTTP200:        http200,
		StatusDist:     statusDist,
		Errors:         errs,
		StartTime:      startTime,
		EndTime:        endTime,
		LatencySamples: durations,
	}
}
