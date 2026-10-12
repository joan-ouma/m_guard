package geolocation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
	Found     bool
	ExpiresAt time.Time
}

type CellResolver struct {
	client             *http.Client
	cacheMu            sync.RWMutex
	cache              map[string]Coordinates
	endpoint           string
	openCellIDEndpoint string
	cacheTTL           time.Duration
}

func NewCellResolver() *CellResolver {
	return &CellResolver{
		client: &http.Client{
			Timeout: 4 * time.Second,
		},
		cache:              make(map[string]Coordinates),
		endpoint:           "https://api.beacondb.net/v1/geolocate",
		openCellIDEndpoint: "https://opencellid.org/ajax/searchCell.php",
		cacheTTL:           24 * time.Hour,
	}
}

// Resolve looks up the geographic coordinates of a GSM cell tower using high-precision dual databases (OpenCelliD + BeaconDB).
func (r *CellResolver) Resolve(ctx context.Context, mcc, mnc, lac, cid int) (float64, float64, bool) {
	if mcc <= 0 || lac <= 0 || cid <= 0 {
		return 0, 0, false
	}

	key := fmt.Sprintf("%d:%d:%d:%d", mcc, mnc, lac, cid)

	r.cacheMu.RLock()
	cached, exists := r.cache[key]
	r.cacheMu.RUnlock()

	now := time.Now()
	if exists && now.Before(cached.ExpiresAt) {
		return cached.Latitude, cached.Longitude, cached.Found
	}

	// 1. Try OpenCelliD direct lookup
	if lat, lon, ok := r.fromOpenCelliD(ctx, mcc, mnc, lac, cid); ok {
		log.Printf("[LBS Geolocation - OpenCelliD] Resolved tower %s -> (%.6f, %.6f)", key, lat, lon)
		r.cacheMu.Lock()
		r.cache[key] = Coordinates{
			Latitude:  lat,
			Longitude: lon,
			Found:     true,
			ExpiresAt: now.Add(r.cacheTTL),
		}
		r.cacheMu.Unlock()
		return lat, lon, true
	}

	// 2. Try BeaconDB lookup
	if lat, lon, ok := r.fromBeaconDB(ctx, mcc, mnc, lac, cid); ok {
		log.Printf("[LBS Geolocation - BeaconDB] Resolved tower %s -> (%.6f, %.6f)", key, lat, lon)
		r.cacheMu.Lock()
		r.cache[key] = Coordinates{
			Latitude:  lat,
			Longitude: lon,
			Found:     true,
			ExpiresAt: now.Add(r.cacheTTL),
		}
		r.cacheMu.Unlock()
		return lat, lon, true
	}

	// Negative cache to prevent hammering
	r.cacheMu.Lock()
	r.cache[key] = Coordinates{Found: false, ExpiresAt: now.Add(10 * time.Minute)}
	r.cacheMu.Unlock()
	return 0, 0, false
}

func (r *CellResolver) fromOpenCelliD(ctx context.Context, mcc, mnc, lac, cid int) (float64, float64, bool) {
	url := fmt.Sprintf("%s?mcc=%d&mnc=%d&lac=%d&cell_id=%d", r.openCellIDEndpoint, mcc, mnc, lac, cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}

	var data struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, false
	}
	if data.Lat != 0 || data.Lon != 0 {
		return data.Lat, data.Lon, true
	}
	return 0, 0, false
}

func (r *CellResolver) fromBeaconDB(ctx context.Context, mcc, mnc, lac, cid int) (float64, float64, bool) {
	type cellTowerRequest struct {
		RadioType         string `json:"radioType"`
		MobileCountryCode int    `json:"mobileCountryCode"`
		MobileNetworkCode int    `json:"mobileNetworkCode"`
		LocationAreaCode  int    `json:"locationAreaCode"`
		CellID            int    `json:"cellId"`
	}

	type geoRequest struct {
		CellTowers []cellTowerRequest `json:"cellTowers"`
	}

	reqBody := geoRequest{
		CellTowers: []cellTowerRequest{
			{
				RadioType:         "gsm",
				MobileCountryCode: mcc,
				MobileNetworkCode: mnc,
				LocationAreaCode:  lac,
				CellID:            cid,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, 0, false
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, 0, false
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "MotoGuard-Fleet/1.0")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
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
		return 0, 0, false
	}

	// Strictly reject IP address fallbacks
	if geoResp.Fallback != "" {
		return 0, 0, false
	}

	if geoResp.Location.Lat == 0 && geoResp.Location.Lng == 0 {
		return 0, 0, false
	}

	return geoResp.Location.Lat, geoResp.Location.Lng, true
}
