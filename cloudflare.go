package main

import (
	"context"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

var (
	cfGraphQLEndpoint = "https://api.cloudflare.com/client/v4/graphql/"
)

type cloudflareResponseColo struct {
	Viewer struct {
		Zones []zoneRespColo `json:"zones"`
	} `json:"viewer"`
}

type zoneRespColo struct {
	ColoGroups []struct {
		Dimensions struct {
			ColoCode string `json:"coloCode"`
			Host     string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
		Count uint64 `json:"count"`
		Sum   struct {
			EdgeResponseBytes uint64 `json:"edgeResponseBytes"`
			Visits            uint64 `json:"visits"`
		} `json:"sum"`
	} `json:"httpRequestsAdaptiveGroups"`

	ZoneTag string `json:"zoneTag"`
}

type cloudflareResponseMonthTotal struct {
	Viewer struct {
		Zones []zoneRespMonthTotal `json:"zones"`
	} `json:"viewer"`
}
type zoneRespMonthTotal struct {
	ColoGroups []struct {
		Dimensions struct {
			Host string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
		Count uint64 `json:"count"`
		Sum   struct {
			EdgeResponseBytes uint64 `json:"edgeResponseBytes"`
			Visits            uint64 `json:"visits"`
		} `json:"sum"`
	} `json:"httpRequestsAdaptiveGroups"`

	ZoneTag string `json:"zoneTag"`
}

func fetchZones() []cloudflare.Zone {
	var api *cloudflare.API
	var err error
	if len(cfgCfAPIToken) > 0 {
		api, err = cloudflare.NewWithAPIToken(cfgCfAPIToken)
	} else {
		api, err = cloudflare.New(cfgCfAPIKey, cfgCfAPIEmail)
	}
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	z, err := api.ListZones(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return z
}

func fetchColoTotals(zoneIDs []string, from time.Time, to time.Time) (*cloudflareResponseColo, error) {
	request := graphql.NewRequest(`
	query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag_in: $zoneIDs }) {
				zoneTag
				httpRequestsAdaptiveGroups(
					limit: $limit
					filter: { datetime_geq: $mintime, datetime_lt: $maxtime }
					) {
						count
						dimensions {
							clientRequestHTTPHost
							coloCode
						}
						sum {
							edgeResponseBytes
							visits
						}
					}
				}
			}
		}
	`)
	if len(cfgCfAPIToken) > 0 {
		request.Header.Set("Authorization", "Bearer "+cfgCfAPIToken)
	} else {
		request.Header.Set("X-AUTH-EMAIL", cfgCfAPIEmail)
		request.Header.Set("X-AUTH-KEY", cfgCfAPIKey)
	}
	request.Var("limit", 9999)
	request.Var("maxtime", from)
	request.Var("mintime", to)
	request.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient(cfGraphQLEndpoint)
	var resp cloudflareResponseColo
	if err := graphqlClient.Run(ctx, request, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func fetchMonthTotals(zoneIDs []string, from time.Time, to time.Time) (*cloudflareResponseMonthTotal, error) {
	request := graphql.NewRequest(`
	query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
		viewer {
			zones(filter: { zoneTag_in: $zoneIDs }) {
				zoneTag
				httpRequestsAdaptiveGroups(
					limit: $limit
					filter: { datetime_geq: $mintime, datetime_lt: $maxtime }
					) {
						count
						dimensions {
							clientRequestHTTPHost
						}
						sum {
							edgeResponseBytes
							visits
						}
					}
				}
			}
		}
	`)
	if len(cfgCfAPIToken) > 0 {
		request.Header.Set("Authorization", "Bearer "+cfgCfAPIToken)
	} else {
		request.Header.Set("X-AUTH-EMAIL", cfgCfAPIEmail)
		request.Header.Set("X-AUTH-KEY", cfgCfAPIKey)
	}
	request.Var("limit", 9999)
	request.Var("maxtime", from)
	request.Var("mintime", to)
	request.Var("zoneIDs", zoneIDs)

	ctx := context.Background()
	graphqlClient := graphql.NewClient(cfGraphQLEndpoint)
	var resp cloudflareResponseMonthTotal
	if err := graphqlClient.Run(ctx, request, &resp); err != nil {
		log.Error(err)
		return nil, err
	}

	return &resp, nil
}

func findZoneName(zones []cloudflare.Zone, ID string) string {
	for _, z := range zones {
		if z.ID == ID {
			return z.Name
		}
	}

	return ""
}

func extractZoneIDs(zones []cloudflare.Zone) []string {
	var IDs []string

	for _, z := range zones {
		IDs = append(IDs, z.ID)
	}

	return IDs
}

func filterNonFreePlanZones(zones []cloudflare.Zone) (filteredZones []cloudflare.Zone) {
	for _, z := range zones {
		if z.Plan.ZonePlanCommon.ID != "0feeeeeeeeeeeeeeeeeeeeeeeeeeeeee" {
			filteredZones = append(filteredZones, z)
		}
	}
	return
}
