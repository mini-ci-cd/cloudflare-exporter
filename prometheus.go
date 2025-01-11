package main

import (
	"fmt"
	"strconv"
	"sync"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricName string

func (mn MetricName) String() string {
	return string(mn)
}

const (
	zoneRequestStatusCountryHostMetricName MetricName = "cloudflare_zone_requests_status_country_host"
)

type MetricsSet map[MetricName]struct{}

func (ms MetricsSet) Has(mn MetricName) bool {
	_, exists := ms[mn]
	return exists
}

func (ms MetricsSet) Add(mn MetricName) {
	ms[mn] = struct{}{}
}

var (
	// Requests

	zoneRequestStatusCountryHost = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: zoneRequestStatusCountryHostMetricName.String(),
		Help: "Count of requests for zone per edge HTTP status per country per host",
	}, []string{"zone", "status", "host", "path"},
	)
)

func buildAllMetricsSet() MetricsSet {
	allMetricsSet := MetricsSet{}
	allMetricsSet.Add(zoneRequestStatusCountryHostMetricName)
	return allMetricsSet
}

func buildDeniedMetricsSet(metricsDenylist []string) (MetricsSet, error) {
	deniedMetricsSet := MetricsSet{}
	allMetricsSet := buildAllMetricsSet()
	for _, metric := range metricsDenylist {
		if !allMetricsSet.Has(MetricName(metric)) {
			return nil, fmt.Errorf("metric %s doesn't exists", metric)
		}
		deniedMetricsSet.Add(MetricName(metric))
	}
	return deniedMetricsSet, nil
}

func mustRegisterMetrics(deniedMetrics MetricsSet) {

	if !deniedMetrics.Has(zoneRequestStatusCountryHostMetricName) {
		prometheus.MustRegister(zoneRequestStatusCountryHost)
	}

}

func fetchZoneAnalytics(zones []cloudflare.Zone, domainNames []string, requestPaths []string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// None of the below referenced metrics are available in the free tier
	if cfgFreeTier {
		return
	}

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	for _, requestPath := range requestPaths {
		r, err := fetchZoneTotals(zoneIDs, domainNames, requestPath)
		if err != nil {
			continue
		}

		for _, z := range r.Viewer.Zones {
			name := findZoneName(zones, z.ZoneTag)
			addHTTPGroups(&z, name)
			addHTTPAdaptiveGroups(&z, name, requestPath)
		}
	}
}

func addHTTPGroups(z *zoneResp, _ string) {
	// Nothing to do.
	if len(z.HTTP1mGroups) == 0 {
		return
	}
	//	zt := z.HTTP1mGroups[0]

}

func addHTTPAdaptiveGroups(z *zoneResp, name string, requestPath string) {

	for _, g := range z.HTTPRequestsEdgeCountryHost {
		zoneRequestStatusCountryHost.With(
			prometheus.Labels{
				"zone":   name,
				"status": strconv.Itoa(int(g.Dimensions.EdgeResponseStatus)),
				"host":   g.Dimensions.ClientRequestHTTPHost,
				"path":   requestPath,
			}).Add(float64(g.Count))
	}

}
