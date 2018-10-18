package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/elastic/hey-apm/compose"
)

const defaultUserAgent = "hey-apm/v2"

var (
	// run options
	runTimeout   = flag.Duration("run", 10*time.Second, "stop run after this duration")
	restDuration = flag.Duration("rest-duration", 100*time.Millisecond, "how long to stop sending data")
	restInterval = flag.Duration("rest-interval", 500*time.Millisecond, "how often to stop sending data")

	// payload options
	numAgents       = flag.Int("c", 3, "concurrent clients")
	numFrames       = flag.Int("f", 1, "number of stacktrace frames per span")
	numSpans        = flag.Int("s", 1, "number of spans")
	numTransactions = flag.Int("t", math.MaxInt64, "number of transactions")

	// http options
	baseUrl            = flag.String("base-url", "http://localhost:8200", "apm-server url")
	requestTimeout     = flag.Duration("request-timeout", 10*time.Second, "request timeout")
	idleTimeout        = flag.Duration("idle-timeout", 3*time.Minute, "idle timeout")
	disableCompression = flag.Bool("disable-compression", false, "")
	disableKeepAlives  = flag.Bool("disable-keepalive", false, "")
	disableRedirects   = flag.Bool("disable-redirects", false, "")
)

func do(parent context.Context, logger *log.Logger, client *http.Client, payloads [][]byte, transactions int) {
	ctx, cancel := context.WithCancel(parent)
	reader, writer := io.Pipe()

	doneWriting := make(chan struct{})
	writes := 0 // payloads
	wrote := 0  // bytes
	go func(w io.WriteCloser) {
		defer close(doneWriting)
		var b = w
		if !*disableCompression {
			defer w.Close()
			b = gzip.NewWriter(w)
			defer b.Close()
		}
		if n, err := b.Write(addNewline(compose.Metadata)); err != nil {
			logger.Println("[error] writing metadata: ", err)
			return
		} else {
			writes++
			wrote += n
		}
		rest := time.After(*restInterval)
		for t := 0; t < transactions; t++ {
			for _, p := range payloads {
				select {
				case <-ctx.Done():
					return
				case <-rest:
					time.Sleep(*restDuration)
					rest = time.After(*restInterval)
				default:
					if n, err := b.Write(p); err != nil {
						logger.Println("[error] writing payload: ", err)
						return
					} else {
						writes++
						wrote += n
					}
				}
			}
		}
	}(writer)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/intake/v2/events", *baseUrl), reader)
	if err != nil {
		logger.Println("[error] creating request:", err)
		return
	}
	req.Header.Add("Content-Type", "application/x-ndjson")
	if !*disableCompression {
		req.Header.Add("Content-Encoding", "gzip")
	}
	// Use the defaultUserAgent unless the Header contains one, which may be blank to not send the header.
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Add("User-Agent", defaultUserAgent)
	}
	rsp, err := client.Do(req)
	cancel()
	if err != nil {
		logger.Println("[error] http client:", err)
		return
	}
	<-doneWriting
	rspBody, _ := ioutil.ReadAll(rsp.Body)
	logger.Printf("[info] after %d writes totaling %d bytes: %s %s\n", writes, wrote, rsp.Status, string(rspBody))
}

func singleTransaction() []byte {
	t := make(map[string]interface{})
	if err := json.Unmarshal(compose.SingleTransaction, &t); err != nil {
		panic(err)
	}
	t["span_count"] = map[string]int{
		"started": *numSpans,
		"dropped": 0,
	}
	t["trace_id"] = "XXX"
	if b, err := json.Marshal(t); err != nil {
		panic(err)
	} else {
		return b
	}
}

func main() {
	flag.Parse()
	ctx, _ := context.WithTimeout(context.Background(), *runTimeout)
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1,
		IdleConnTimeout:     *idleTimeout,
		DisableCompression:  *disableCompression,
		DisableKeepAlives:   *disableKeepAlives,
	}
	client := &http.Client{
		Timeout:   *requestTimeout,
		Transport: tr,
	}
	if *disableRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	span := addNewline(compose.NdjsonWrapObj("span", compose.V2Span(*numFrames)))
	payloads := make([][]byte, 1+*numSpans)
	payloads[0] = addNewline(compose.NdjsonWrapObj("transaction", singleTransaction()))
	for i := 1; i <= *numSpans; i++ {
		payloads[i] = span
	}

	var wg sync.WaitGroup
	extraT := *numTransactions % *numAgents
	agentT := *numTransactions / *numAgents
	if agentT == 0 {
		log.Printf("[warn] not enough transactions (%d) to go around to %d agents", *numTransactions, *numAgents)
		*numAgents = 1
	}
	wg.Add(*numAgents)
	for i := 0; i < *numAgents; i++ {
		logger := log.New(os.Stderr, fmt.Sprintf("[agent %d] ", i), log.Ldate|log.Ltime|log.Lshortfile)

		go func(countT int) {
			logger.Println("[debug] starting new connection")
			do(ctx, logger, client, payloads, countT)
			wg.Done()
		}(agentT + extraT)
		extraT = 0
	}
	wg.Wait()
}

func addNewline(p []byte) []byte {
	return append(p, '\n')
}
