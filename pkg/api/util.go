package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/notque/netflow-api/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// utility functionality

//VersionData is used by version advertisement handlers.
type VersionData struct {
	Status string            `json:"status"`
	ID     string            `json:"id"`
	Links  []versionLinkData `json:"links"`
}

//versionLinkData is used by version advertisement handlers, as part of the
//VersionData struct.
type versionLinkData struct {
	URL      string `json:"href"`
	Relation string `json:"rel"`
	Type     string `json:"type,omitempty"`
}

//ReturnJSON is a convenience function for HTTP handlers returning JSON data.
//The `code` argument specifies the HTTP response code, usually 200.
func ReturnJSON(w http.ResponseWriter, code int, data interface{}) {
	payload, err := json.MarshalIndent(&data, "", "  ")
	// Replaces & symbols properly in json within urls due to Elasticsearch
	payload = bytes.Replace(payload, []byte("\\u0026"), []byte("&"), -1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(payload)
	if err != nil {
		util.LogDebug("Issue with writing payload when returning Json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//ReturnError produces an error response with HTTP status code 500 if the given
//error is non-nil. Otherwise, nothing is done and false is returned.
func ReturnError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	http.Error(w, err.Error(), 500)
	return true
}

var authErrorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "netflow-api_logon_errors_count", Help: "Number of logon errors occurred"})
var authFailuresCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "netflow-api_logon_failures_count", Help: "Number of logon attempts failed due to wrong credentials"})
var storageErrorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "netflow-api_storage_errors_count", Help: "Number of technical errors occurred when accessing underlying storage (i.e. Elasticsearch)"})

func init() {
	prometheus.MustRegister(authErrorsCounter, authFailuresCounter, storageErrorsCounter)
}

func gaugeInflight(handler http.Handler) http.Handler {
	inflightGauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "netflow-api_requests_inflight", Help: "Number of inflight HTTP requests served by netflow-api"})
	prometheus.MustRegister(inflightGauge)

	return promhttp.InstrumentHandlerInFlight(inflightGauge, handler)
}

func observeDuration(handlerFunc http.HandlerFunc, handler string) http.HandlerFunc {
	durationSummary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{Name: "netflow-api_request_duration_seconds", Help: "Duration/latency of a netflow-api request", ConstLabels: prometheus.Labels{"handler": handler}}, nil)
	prometheus.MustRegister(durationSummary)

	return promhttp.InstrumentHandlerDuration(durationSummary, handlerFunc)
}

func observeResponseSize(handlerFunc http.HandlerFunc, handler string) http.HandlerFunc {
	durationSummary := prometheus.NewSummaryVec(prometheus.SummaryOpts{Name: "netflow-api_response_size_bytes", Help: "Size of the netflow-api response (e.g. to a query)", ConstLabels: prometheus.Labels{"handler": handler}}, nil)
	prometheus.MustRegister(durationSummary)

	return promhttp.InstrumentHandlerResponseSize(durationSummary, http.HandlerFunc(handlerFunc)).ServeHTTP
}
