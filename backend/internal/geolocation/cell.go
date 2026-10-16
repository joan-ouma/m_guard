package geolocation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"motoguard-backend/internal/domain"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
	Accuracy  float64
	Found     bool
	ExpiresAt time.Time
}

type CellResolver struct {
	client   *http.Client
	cacheMu  sync.RWMutex
	cache    map[string]Coordinates
	endpoint string
	cacheTTL time.Duration
}

func NewCellResolver() *CellResolver {
	return &CellResolver{
		client: &http.Client{
			Timeout: 4 * time.Second,
		},
		cache:    make(map[string]Coordinates),
		endpoint: "https://api.beacondb.net/v1/geolocate",
		cacheTTL: 24 * time.Hour,
	}
}

func buildCellCacheKey(mcc, mnc int, cells []domain.CellTower) string {
	if len(cells) == 1 {
		return fmt.Sprintf("single:%d:%d:%d:%d", mcc, mnc, cells[0].LAC, cells[0].CID)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("multi:%d:%d", mcc, mnc))
	for _, c := range cells {
		if c.LAC > 0 && c.CID > 0 {
			sb.WriteString(fmt.Sprintf("|%d:%d", c.LAC, c.CID))
		}
	}
	return sb.String()
}

// Resolve single serving cell
func (r *CellResolver) Resolve(ctx context.Context, mcc, mnc, lac, cid int) (float64, float64, bool) {
	lat, lng, _, ok := r.ResolveMulti(ctx, mcc, mnc, []domain.CellTower{
		{LAC: lac, CID: cid, RSSIDBm: -75},
	})
	return lat, lng, ok
}

// ResolveMulti performs multi-cell resolution using BeaconDB and weighted multilateration
func (r *CellResolver) ResolveMulti(ctx context.Context, mcc, mnc int, cells []domain.CellTower) (float64, float64, float64, bool) {
	if mcc <= 0 || len(cells) == 0 {
		return 0, 0, 0, false
	}

	primary := cells[0]
	if primary.LAC <= 0 || primary.CID <= 0 {
		return 0, 0, 0, false
	}

	cacheKey := buildCellCacheKey(mcc, mnc, cells)

	r.cacheMu.RLock()
	cached, exists := r.cache[cacheKey]
	r.cacheMu.RUnlock()

	now := time.Now()
	if exists && now.Before(cached.ExpiresAt) {
		if !cached.Found {
			return 0, 0, 0, false
		}
		return cached.Latitude, cached.Longitude, cached.Accuracy, true
	}

	// 1. Resolve via multi-cell cloud geolocator (BeaconDB / Google-compatible schema)
	lat, lng, accuracy, ok := r.fromMultiCellCloud(ctx, mcc, mnc, cells)
	if ok {
		slog.Info("LBS Multi-cell resolved", "towers", len(cells), "lat", lat, "lng", lng, "accuracy", accuracy)
		r.cacheMu.Lock()
		r.cache[cacheKey] = Coordinates{
			Latitude:  lat,
			Longitude: lng,
			Accuracy:  accuracy,
			Found:     true,
			ExpiresAt: now.Add(r.cacheTTL),
		}
		// When resolving a single tower, also cache its specific tower position
		if len(cells) == 1 {
			singleKey := fmt.Sprintf("single:%d:%d:%d:%d", mcc, mnc, primary.LAC, primary.CID)
			r.cache[singleKey] = Coordinates{
				Latitude:  lat,
				Longitude: lng,
				Accuracy:  accuracy,
				Found:     true,
				ExpiresAt: now.Add(r.cacheTTL),
			}
		}
		r.cacheMu.Unlock()
		return lat, lng, accuracy, true
	}

	// 2. Multilateration fallback from known cached individual towers if available
	lat, lng, accuracy, ok = r.triangulateKnownCells(mcc, mnc, cells)
	if ok {
		slog.Info("LBS Multilateration computed from local cell cache", "towers", len(cells), "lat", lat, "lng", lng, "accuracy", accuracy)
		return lat, lng, accuracy, true
	}

	r.cacheMu.Lock()
	r.cache[cacheKey] = Coordinates{Found: false, ExpiresAt: now.Add(10 * time.Minute)}
	r.cacheMu.Unlock()
	return 0, 0, 0, false
}

func (r *CellResolver) triangulateKnownCells(mcc, mnc int, cells []domain.CellTower) (float64, float64, float64, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	type resolvedPoint struct {
		lat    float64
		lng    float64
		weight float64
	}

	var points []resolvedPoint
	for _, c := range cells {
		kSingle := fmt.Sprintf("single:%d:%d:%d:%d", mcc, mnc, c.LAC, c.CID)
		coord, exists := r.cache[kSingle]
		if !exists || !coord.Found {
			kLegacy := fmt.Sprintf("%d:%d:%d:%d", mcc, mnc, c.LAC, c.CID)
			coord, exists = r.cache[kLegacy]
		}
		if !exists || !coord.Found {
			continue
		}

		dbm := float64(c.RSSIDBm)
		if dbm == 0 {
			dbm = -85
		}
		weight := math.Pow(10, (dbm+115.0)/20.0)
		if weight < 0.1 {
			weight = 0.1
		}
		points = append(points, resolvedPoint{
			lat:    coord.Latitude,
			lng:    coord.Longitude,
			weight: weight,
		})
	}

	if len(points) == 0 {
		return 0, 0, 0, false
	}

	var sumLat, sumLng, sumWeight float64
	for _, p := range points {
		sumLat += p.lat * p.weight
		sumLng += p.lng * p.weight
		sumWeight += p.weight
	}

	centroidLat := sumLat / sumWeight
	centroidLng := sumLng / sumWeight

	var maxDistKM float64
	for _, p := range points {
		d := haversineDistanceKM(centroidLat, centroidLng, p.lat, p.lng)
		if d > maxDistKM {
			maxDistKM = d
		}
	}

	accuracyMeters := maxDistKM * 1000.0
	if accuracyMeters < 80.0 {
		accuracyMeters = 80.0
	}
	if accuracyMeters > 2500.0 {
		accuracyMeters = 2500.0
	}

	return centroidLat, centroidLng, accuracyMeters, true
}

func (r *CellResolver) fromMultiCellCloud(ctx context.Context, mcc, mnc int, cells []domain.CellTower) (float64, float64, float64, bool) {
	type cellRequest struct {
		RadioType         string `json:"radioType"`
		MobileCountryCode int    `json:"mobileCountryCode"`
		MobileNetworkCode int    `json:"mobileNetworkCode"`
		LocationAreaCode  int    `json:"locationAreaCode"`
		CellID            int    `json:"cellId"`
		SignalStrength    int    `json:"signalStrength,omitempty"`
	}

	type geoRequest struct {
		CellTowers []cellRequest `json:"cellTowers"`
	}

	reqTowers := make([]cellRequest, 0, len(cells))
	for _, c := range cells {
		if c.LAC <= 0 || c.CID <= 0 {
			continue
		}
		reqTowers = append(reqTowers, cellRequest{
			RadioType:         "gsm",
			MobileCountryCode: mcc,
			MobileNetworkCode: mnc,
			LocationAreaCode:  c.LAC,
			CellID:            c.CID,
			SignalStrength:    c.RSSIDBm,
		})
	}

	if len(reqTowers) == 0 {
		return 0, 0, 0, false
	}

	reqBody := geoRequest{CellTowers: reqTowers}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, 0, 0, false
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, 0, 0, false
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "MotoGuard-Fleet/1.0 (LBS-MultiCell)")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return 0, 0, 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, false
	}

	var geoResp struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		Accuracy float64 `json:"accuracy"`
		Fallback string  `json:"fallback"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		return 0, 0, 0, false
	}

	if geoResp.Location.Lat == 0 && geoResp.Location.Lng == 0 {
		return 0, 0, 0, false
	}

	accuracy := geoResp.Accuracy
	if accuracy <= 0 {
		if geoResp.Fallback != "" {
			accuracy = 2500.0
		} else {
			accuracy = 350.0
		}
	}

	return geoResp.Location.Lat, geoResp.Location.Lng, accuracy, true
}

func haversineDistanceKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}
