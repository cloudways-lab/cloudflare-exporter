package main

import (
	"sync"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Requests
	zoneColocationVisits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_visits",
		Help: "Total visits per colocation",
	}, []string{"zone", "colocation", "host"},
	)

	zoneHostMonthTotalVisits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_month_visits_total",
		Help: "Total visits per host",
	}, []string{"zone", "host"},
	)

	zoneColocationEdgeResponseBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_edge_response_bytes",
		Help: "Edge response bytes per colocation",
	}, []string{"zone", "colocation", "host"},
	)

	zoneHostMonthTotalEdgeResponseBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_month_edge_response_bytes_total",
		Help: "Edge response bytes per host",
	}, []string{"zone", "host"},
	)

	zoneColocationRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_colocation_requests_total",
		Help: "Total requests per colocation",
	}, []string{"zone", "colocation", "host"},
	)

	zoneHostMonthRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudflare_zone_month_requests_total",
		Help: "Total requests per colocation",
	}, []string{"zone", "host"},
	)
)

func fetchZoneColocationAnalytics(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// // Colocation metrics are not available in non-enterprise zones
	// if cfgFreeTier {
	// 	return
	// }

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	now := time.Now().Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	r, err := fetchColoTotals(zoneIDs, now, now1mAgo)
	if err != nil {
		return
	}

	for _, z := range r.Viewer.Zones {

		cg := z.ColoGroups
		name := findZoneName(zones, z.ZoneTag)
		for _, c := range cg {
			zoneColocationVisits.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Sum.Visits))
			zoneColocationEdgeResponseBytes.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Sum.EdgeResponseBytes))
			zoneColocationRequestsTotal.With(prometheus.Labels{"zone": name, "colocation": c.Dimensions.ColoCode, "host": c.Dimensions.Host}).Add(float64(c.Count))
		}
	}
}

func fetchZoneCalendarMonthTotals(zones []cloudflare.Zone, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	zoneIDs := extractZoneIDs(filterNonFreePlanZones(zones))
	if len(zoneIDs) == 0 {
		return
	}

	now := time.Now()
	// Fake last month for testing
	// now = time.Date(now.Year(), now.Month()-1, now.Day(), 0, 0, 0, 0, now.Location())

	now = now.Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	r, err := fetchMonthTotals(zoneIDs, now, firstOfMonth)
	if err != nil {
		return
	}

	for _, z := range r.Viewer.Zones {
		cg := z.ColoGroups
		name := findZoneName(zones, z.ZoneTag)
		for _, c := range cg {
			zoneHostMonthTotalVisits.With(prometheus.Labels{"zone": name, "host": c.Dimensions.Host}).Add(float64(c.Sum.Visits))
			zoneHostMonthTotalEdgeResponseBytes.With(prometheus.Labels{"zone": name, "host": c.Dimensions.Host}).Add(float64(c.Sum.EdgeResponseBytes))
			zoneHostMonthRequestsTotal.With(prometheus.Labels{"zone": name, "host": c.Dimensions.Host}).Add(float64(c.Count))
		}
	}
}
