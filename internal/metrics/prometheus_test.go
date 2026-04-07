package metrics_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/metrics"
	"time"
)

func newTestPrometheus(handler http.Handler) (*metrics.PrometheusProvider, *httptest.Server) {
	srv := httptest.NewServer(handler)
	provider, _ := metrics.NewPrometheusProvider(srv.URL, nil)
	return provider, srv
}

func TestAvailable_Ready(t *testing.T) {
	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/-/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	testza.AssertTrue(t, provider.Available())
}

func TestAvailable_FallbackToConfig(t *testing.T) {
	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status/config" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	testza.AssertTrue(t, provider.Available())
}

func TestAvailable_Unreachable(t *testing.T) {
	provider, _ := metrics.NewPrometheusProvider("http://127.0.0.1:1", nil)
	testza.AssertFalse(t, provider.Available())
}

func TestQueryRange_ParsesMatrixResponse(t *testing.T) {
	var receivedQuery, receivedStart, receivedEnd, receivedStep string

	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("query")
		receivedStart = r.URL.Query().Get("start")
		receivedEnd = r.URL.Query().Get("end")
		receivedStep = r.URL.Query().Get("step")

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "matrix",
				"result": []map[string]interface{}{
					{
						"metric": map[string]string{"container": "nginx"},
						"values": [][]interface{}{
							{1700000000.0, "0.5"},
							{1700000015.0, "0.7"},
						},
					},
					{
						"metric": map[string]string{"container": "sidecar"},
						"values": [][]interface{}{
							{1700000000.0, "0.1"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	start := time.Unix(1700000000, 0)
	end := time.Unix(1700000060, 0)
	step := 15 * time.Second

	series, err := provider.QueryRange(context.Background(), "test_query", start, end, step)
	testza.AssertNoError(t, err)
	testza.AssertLen(t, series, 2)

	testza.AssertEqual(t, "1700000000", receivedStart)
	testza.AssertEqual(t, "1700000060", receivedEnd)
	testza.AssertEqual(t, "15", receivedStep)
	testza.AssertEqual(t, "test_query", receivedQuery)

	testza.AssertEqual(t, "nginx", series[0].Labels["container"])
	testza.AssertLen(t, series[0].Points, 2)
	testza.AssertEqual(t, int64(1700000000), series[0].Points[0].Timestamp)
	testza.AssertEqual(t, 0.5, series[0].Points[0].Value)
	testza.AssertEqual(t, int64(1700000015), series[0].Points[1].Timestamp)
	testza.AssertEqual(t, 0.7, series[0].Points[1].Value)

	testza.AssertEqual(t, "sidecar", series[1].Labels["container"])
	testza.AssertLen(t, series[1].Points, 1)
	testza.AssertEqual(t, 0.1, series[1].Points[0].Value)
}

func TestQueryInstant_ParsesVectorResponse(t *testing.T) {
	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{
						"metric": map[string]string{"container": "app"},
						"value":  []interface{}{1700000000.0, "0.25"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	resp, err := provider.QueryInstant(context.Background(), "core.v1.pods", "default", "my-pod")
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, resp)
	testza.AssertTrue(t, len(resp.Metrics) > 0)
}

func TestQueryRange_ErrorOnNon200(t *testing.T) {
	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "error",
			"errorType": "bad_data",
			"error":     "invalid query",
		})
	}))
	defer srv.Close()

	_, err := provider.QueryRange(context.Background(), "bad", time.Now(), time.Now(), time.Second)
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "bad_data")
	testza.AssertContains(t, err.Error(), "invalid query")
}

func TestQueryRange_ErrorOnMalformedJSON(t *testing.T) {
	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := provider.QueryRange(context.Background(), "q", time.Now(), time.Now(), time.Second)
	testza.AssertNotNil(t, err)
}

func TestQueryRange_ErrorOnHTTP500(t *testing.T) {
	provider, srv := newTestPrometheus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	_, err := provider.QueryRange(context.Background(), "q", time.Now(), time.Now(), time.Second)
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "500")
}
