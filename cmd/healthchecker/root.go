package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	grpcCode "google.golang.org/grpc/codes"
)

var rootCmd = &cobra.Command{
	Use:   "healthchecker <endpoint[,endpoint[,...]]>",
	Short: "prometheus exporter for health checking gRPC server",
	Args:  cobra.ExactArgs(1),
	Run:   runRootCmd,
}

func init() {
	rootCmd.Flags().StringSliceP("header", "H", nil, "Additional headers to be sent in the health checker request (Example: -H Authorization: Bearer $TOKEN)")
	rootCmd.Flags().StringP("path", "p", "", "Path to the gRPC server service (Example: /blockmeta.v1.BlockMetaService/GetBlock)")
	rootCmd.Flags().Duration("lookup_interval", time.Second*15, "endpoints will be requested at this interval")
	rootCmd.Flags().String("request-body-hex", "0000000000", "request body value in hex format")
	rootCmd.Flags().String("listen-addr", ":9102", "prometheus exporter listen address")
}

func runRootCmd(cmd *cobra.Command, args []string) {
	endpoints := strings.Split(args[0], ",")
	listenAddr := mustGetString(cmd, "listen-addr")
	path := mustGetString(cmd, "path")
	additionalHeaders := mustGetStringSlice(cmd, "header")
	lookUpInterval := mustGetDuration(cmd, "lookup_interval")
	requestBodyHex := mustGetHex(cmd, "request-body-hex")

	go launchPoller(endpoints, path, additionalHeaders, requestBodyHex, lookUpInterval)

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(status)
	promReg.MustRegister(requestDurationMs)

	if err := runPrometheusExporter(promReg, listenAddr); err != nil {
		zlog.Error("running prometheus exporter", zap.Error(err))
		os.Exit(1)
	}
}

var status = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "request_healthcheck_status", Help: "Either 1 for successful request, or 0 for failure"}, []string{"endpoint"})
var requestDurationMs = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "request_healthcheck_duration_ms", Help: "Request full processing time in millisecond"}, []string{"endpoint"})

func runPrometheusExporter(promReg *prometheus.Registry, listenAddr string) error {
	handler := promhttp.HandlerFor(
		promReg,
		promhttp.HandlerOpts{
			EnableOpenMetrics: false,
		})

	serve := http.Server{Handler: handler, Addr: listenAddr}

	zlog.Info("listening on the address", zap.String("addr", listenAddr))
	if err := serve.ListenAndServe(); err != nil {
		zlog.Info("can't listen on the metrics endpoint", zap.Error(err))
		return err
	}

	return nil
}
func launchPoller(endpoints []string, path string, additionalHeaders []string, requestBodyHex []byte, lookUpInterval time.Duration) {
	for {
		for _, endpoint := range endpoints {
			go requestGRPCServer(endpoint, path, additionalHeaders, requestBodyHex)
		}
		time.Sleep(lookUpInterval)
	}
}

func requestGRPCServer(endpoint, path string, additionalHeaders []string, requestBodyHex []byte) {
	zlog.Info("Requesting server", zap.String("endpoint", endpoint))
	client := &http.Client{}

	requestBodyBuffer := bytes.NewBuffer(requestBodyHex)
	req, err := http.NewRequest("POST", "https://"+endpoint+path, requestBodyBuffer)
	if err != nil {
		zlog.Error("creating request", zap.Error(err))
		os.Exit(1)
	}

	req.Header.Add("Content-Type", "application/grpc")
	req.Header.Add("TE", "trailers")

	for _, header := range additionalHeaders {
		headerParts := strings.Split(header, ":")
		req.Header.Add(headerParts[0], headerParts[1])
	}

	begin := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		zlog.Error("sending request", zap.Error(err))
		os.Exit(1)
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		zlog.Error("reading response body", zap.Error(err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	grpcStatusCode, err := strconv.Atoi(resp.Trailer.Get("Grpc-Status"))
	if err != nil {
		zlog.Error("converting grpc status to int", zap.Error(err))
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK || grpcStatusCode != int(grpcCode.OK) {
		zlog.Error("request status", zap.Int("status_code", resp.StatusCode), zap.String("grpc_status", resp.Trailer.Get("Grpc-Status")), zap.String("grpc_message", resp.Trailer.Get("Grpc-Message")))
		markFailure(endpoint, begin)
	}

	markSuccess(endpoint, begin)
}

func markSuccess(endpoint string, begin time.Time) {
	status.With(prometheus.Labels{"endpoint": endpoint}).Set(1)
	requestDurationMs.With(prometheus.Labels{"endpoint": endpoint}).Set(float64(time.Since(begin).Milliseconds()))
}

func markFailure(endpoint string, begin time.Time) {
	status.With(prometheus.Labels{"endpoint": endpoint}).Set(0)
	requestDurationMs.With(prometheus.Labels{"endpoint": endpoint}).Set(float64(time.Since(begin).Milliseconds()))
}
