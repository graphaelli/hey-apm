package models

import (
	"time"
)

// Input holds all the parameters given to a load test work.
// Most parameters describe a workload pattern, others are required to create performance reports.
type Input struct {

	// Whether or not this object will be processed by the `benchmark` package
	IsBenchmark bool `json:"-"`
	// Number of days to look back for regressions (only if IsBenchmark is true)
	RegressionDays string `json:"-"`
	// Acceptable performance decrease without being considered as regressions, as a percentage
	// (only if IsBenchmark is true)
	RegressionMargin float64 `json:"-"`

	// URL of the APM Server under test
	ApmServerUrl string `json:"apm_url"`
	// Secret token of the APM Server under test
	ApmServerSecret string `json:"-"`
	// API Key for communication between APM Server and the Go agent
	APIKey string `json:"-"`
	// If true, it will index the performance report of a run in ElasticSearch
	SkipIndexReport bool `json:"-"`
	// URL of the Elasticsearch instance used for indexing the performance report
	ElasticsearchUrl string `json:"-"`
	// <username:password> of the Elasticsearch instance used for indexing the performance report
	ElasticsearchAuth string `json:"-"`
	// URL of the Elasticsearch instance used by APM Server
	ApmElasticsearchUrl string `json:"elastic_url,omitempty"`
	// <username:password> of the Elasticsearch instance used by APM Server
	ApmElasticsearchAuth string `json:"-"`
	// Service name passed to the tracer
	ServiceName string `json:"service_name,omitempty"`

	// Run timeout of the performance test (ends the test when reached)
	RunTimeout time.Duration `json:"run_timeout"`
	// Timeout for flushing the workload to APM Server
	FlushTimeout time.Duration `json:"flush_timeout"`
	// Number of Instances that are creating load
	Instances int `json:"instances"`
	// DelayMillis is the maximum amount of milliseconds to wait per instance before starting it,
	// can be used to add some randomness for producing load
	DelayMillis int `json:"delay_millis"`
	// Frequency at which the tracer will generate transactions
	TransactionFrequency time.Duration `json:"transaction_generation_frequency"`
	// Maximum number of transactions to push to the APM Server (ends the test when reached)
	TransactionLimit int `json:"transaction_generation_limit"`
	// Maximum number of spans per transaction
	SpanMaxLimit int `json:"spans_generated_max_limit"`
	// Minimum number of spans per transaction
	SpanMinLimit int `json:"spans_generated_min_limit"`
	// Frequency at which the tracer will generate errors
	ErrorFrequency time.Duration `json:"error_generation_frequency"`
	// Maximum number of errors to push to the APM Server (ends the test when reached)
	ErrorLimit int `json:"error_generation_limit"`
	// Maximum number of stacktrace frames per error
	ErrorFrameMaxLimit int `json:"error_generation_frames_max_limit"`
	// Minimum number of stacktrace frames per error
	ErrorFrameMinLimit int `json:"error_generation_frames_min_limit"`
}

func (in Input) WithErrors(limit int, freq time.Duration) Input {
	in.ErrorLimit = limit
	in.ErrorFrequency = freq
	return in
}

func (in Input) WithFrames(f int) Input {
	in.ErrorFrameMaxLimit = f
	in.ErrorFrameMinLimit = f
	return in
}

func (in Input) WithTransactions(limit int, freq time.Duration) Input {
	in.TransactionLimit = limit
	in.TransactionFrequency = freq
	return in
}

func (in Input) WithSpans(s int) Input {
	in.SpanMaxLimit = s
	in.SpanMinLimit = s
	return in
}
