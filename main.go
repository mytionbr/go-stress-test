package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
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
	AvgLatencyMS   float64
	MinLatencyMS   float64
	P95LatencyMS   float64
	MaxLatencyMS   float64
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
				fmt.Println("Erro na requisição:", err)
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

	rep := buildReport(url, total, concurrency, startTime, results)

	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintln(os.Stderr, "erro ao gerar JSON:", err)
			os.Exit(1)
		}
		return
	}

	printReport(rep)
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

	avg, min, p95, max := latencyStats(durations)

	return report{
		URL:            url,
		TotalRequests:  total,
		Concurrency:    concurrency,
		TotalTime:      totalTime,
		RequestsPerSec: float64(total) / totalTime.Seconds(),
		HTTP200:        http200,
		StatusDist:     statusDist,
		Errors:         errs,
		AvgLatencyMS:   avg,
		MinLatencyMS:   min,
		P95LatencyMS:   p95,
		MaxLatencyMS:   max,
		StartTime:      startTime,
		EndTime:        endTime,
		LatencySamples: durations,
	}
}

func latencyStats(durations []time.Duration) (avg, min, p95, max float64) {
	if len(durations) == 0 {
		return 0, 0, 0, 0
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var sum time.Duration
	minDur := durations[0]
	maxDur := durations[len(durations)-1]
	for _, d := range durations {
		sum += d
	}
	avg = float64(sum.Milliseconds()) / float64(len(durations))
	min = float64(minDur.Milliseconds())
	max = float64(maxDur.Milliseconds())
	idx := int(float64(len(durations))*0.95) - 1
	if idx < 0 {
		idx = 0
	}
	p95 = float64(durations[idx].Milliseconds())
	return
}

func printReport(r report) {
	fmt.Println("==== Relatório de Teste de Carga ====")
	fmt.Printf("URL:                 %s\n", r.URL)
	fmt.Printf("Requests Totais:     %d\n", r.TotalRequests)
	fmt.Printf("Concorrência:        %d\n", r.Concurrency)
	fmt.Printf("Tempo Total:         %s\n", r.TotalTime)
	fmt.Printf("RPS (aprox):         %.2f req/s\n", r.RequestsPerSec)
	fmt.Printf("HTTP 200:            %d\n", r.HTTP200)
	fmt.Printf("Erros (timeout/etc): %d\n", r.Errors)
	fmt.Println("\nDistribuição de Status HTTP:")
	// ordenar chaves
	keys := make([]int, 0, len(r.StatusDist))
	for k := range r.StatusDist {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("  %d: %d\n", k, r.StatusDist[k])
	}
	fmt.Println("\nLatências (ms):")
	fmt.Printf("  Média:  %.2f\n", r.AvgLatencyMS)
	fmt.Printf("  Mínima: %.2f\n", r.MinLatencyMS)
	fmt.Printf("  P95:    %.2f\n", r.P95LatencyMS)
	fmt.Printf("  Máxima: %.2f\n", r.MaxLatencyMS)
	fmt.Printf("\nInício: %s\nFim:    %s\n", r.StartTime.Format(time.RFC3339), r.EndTime.Format(time.RFC3339))
}
