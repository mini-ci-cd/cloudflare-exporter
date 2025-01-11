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
	newGraphQLClient  = graphql.NewClient
)

type cloudflareResponse struct {
	Viewer struct {
		Zones []zoneResp `json:"zones"`
	} `json:"viewer"`
}

type zoneResp struct {
	HTTP1mGroups []struct {
		Dimensions struct {
			Datetime string `json:"datetime"`
		} `json:"dimensions"`
		Unique struct {
			Uniques uint64 `json:"uniques"`
		} `json:"uniq"`
		Sum struct {
			Bytes          uint64 `json:"bytes"`
			CachedBytes    uint64 `json:"cachedBytes"`
			CachedRequests uint64 `json:"cachedRequests"`
			Requests       uint64 `json:"requests"`
			BrowserMap     []struct {
				PageViews       uint64 `json:"pageViews"`
				UaBrowserFamily string `json:"uaBrowserFamily"`
			} `json:"browserMap"`
			ClientHTTPVersion []struct {
				Protocol string `json:"clientHTTPProtocol"`
				Requests uint64 `json:"requests"`
			} `json:"clientHTTPVersionMap"`
			ClientSSL []struct {
				Protocol string `json:"clientSSLProtocol"`
			} `json:"clientSSLMap"`
			ContentType []struct {
				Bytes                   uint64 `json:"bytes"`
				Requests                uint64 `json:"requests"`
				EdgeResponseContentType string `json:"edgeResponseContentTypeName"`
			} `json:"contentTypeMap"`
			Country []struct {
				Bytes             uint64 `json:"bytes"`
				ClientCountryName string `json:"clientCountryName"`
				Requests          uint64 `json:"requests"`
				Threats           uint64 `json:"threats"`
			} `json:"countryMap"`
			EncryptedBytes    uint64 `json:"encryptedBytes"`
			EncryptedRequests uint64 `json:"encryptedRequests"`
			IPClass           []struct {
				Type     string `json:"ipType"`
				Requests uint64 `json:"requests"`
			} `json:"ipClassMap"`
			PageViews      uint64 `json:"pageViews"`
			ResponseStatus []struct {
				EdgeResponseStatus int    `json:"edgeResponseStatus"`
				Requests           uint64 `json:"requests"`
			} `json:"responseStatusMap"`
			ThreatPathing []struct {
				Name     string `json:"threatPathingName"`
				Requests uint64 `json:"requests"`
			} `json:"threatPathingMap"`
			Threats uint64 `json:"threats"`
		} `json:"sum"`
	} `json:"httpRequests1mGroups"`

	HTTPRequestsEdgeCountryHost []struct {
		Count      uint64 `json:"count"`
		Dimensions struct {
			EdgeResponseStatus    uint16 `json:"edgeResponseStatus"`
			ClientRequestHTTPHost string `json:"clientRequestHTTPHost"`
		} `json:"dimensions"`
	} `json:"httpRequestsEdgeCountryHost"`

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

func fetchZoneTotals(zoneIDs []string, domainNames []string, requestPath string) (*cloudflareResponse, error) {
	now := time.Now().Add(-time.Duration(cfgScrapeDelay) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo := now.Add(-60 * time.Second)

	request := graphql.NewRequest(`
query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!) {
	viewer {
		zones(filter: { zoneTag_in: $zoneIDs }) {
			zoneTag

httpRequestsEdgeCountryHost: httpRequestsAdaptiveGroups(limit: $limit, filter:
  {
    datetime_geq: $mintime,
    datetime_lt: $maxtime,
    clientRequestPath_like: $requestPath,
    clientRequestHTTPHost_in: $domainNames
  })
  {
				count
				dimensions {
					edgeResponseStatus
					clientRequestHTTPHost
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
	request.Var("maxtime", now)
	request.Var("mintime", now1mAgo)
	request.Var("zoneIDs", zoneIDs)
	request.Var("domainNames", domainNames)
	request.Var("requestPath", requestPath)

	ctx := context.Background()
	graphqlClient := newGraphQLClient(cfGraphQLEndpoint)

	var resp cloudflareResponse
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
