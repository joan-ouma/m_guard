package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AddressInfo struct {
	FormattedAddress string
	SubCounty        string
	County           string
}

type ReverseGeocoder struct {
	client     *http.Client
	cacheMu    sync.RWMutex
	cache      map[string]AddressInfo
	cacheTTL   time.Duration
	lastNomMu  sync.Mutex
	lastNomReq time.Time
}

func NewReverseGeocoder() *ReverseGeocoder {
	return &ReverseGeocoder{
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		cache:    make(map[string]AddressInfo),
		cacheTTL: 24 * time.Hour,
	}
}

func (g *ReverseGeocoder) ReverseGeocode(ctx context.Context, lat, lng float64) AddressInfo {
	if lat == 0 && lng == 0 {
		return AddressInfo{
			FormattedAddress: "",
			SubCounty:        "",
			County:           "",
		}
	}

	key := fmt.Sprintf("%.3f:%.3f", lat, lng)

	g.cacheMu.RLock()
	cached, ok := g.cache[key]
	g.cacheMu.RUnlock()
	if ok {
		return cached
	}

	// 1. Primary: BigDataCloud (designed for mobile/device reverse geocoding)
	info, err := g.fromBigDataCloud(ctx, lat, lng)
	if err == nil && info.FormattedAddress != "" {
		g.cacheMu.Lock()
		g.cache[key] = info
		g.cacheMu.Unlock()
		return info
	}

	// 2. Throttled Fallback: Nominatim (strict 1.5s throttle to comply with OSM policy)
	if g.canCallNominatim() {
		info, err = g.fromNominatim(ctx, lat, lng)
		if err == nil && info.FormattedAddress != "" {
			g.cacheMu.Lock()
			g.cache[key] = info
			g.cacheMu.Unlock()
			return info
		}
	}

	empty := AddressInfo{
		FormattedAddress: "",
		SubCounty:        "",
		County:           "",
	}
	g.cacheMu.Lock()
	g.cache[key] = empty
	g.cacheMu.Unlock()
	return empty
}

func (g *ReverseGeocoder) canCallNominatim() bool {
	g.lastNomMu.Lock()
	defer g.lastNomMu.Unlock()
	now := time.Now()
	if now.Sub(g.lastNomReq) < 1500*time.Millisecond {
		return false
	}
	g.lastNomReq = now
	return true
}

func (g *ReverseGeocoder) fromNominatim(ctx context.Context, lat, lng float64) (AddressInfo, error) {
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%.6f&lon=%.6f&format=json&zoom=18", lat, lng)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return AddressInfo{}, err
	}
	req.Header.Set("User-Agent", "MotoGuard-Fleet/1.0 (ReverseGeocoder)")

	resp, err := g.client.Do(req)
	if err != nil {
		return AddressInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AddressInfo{}, fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	var data struct {
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
		Address     struct {
			Road          string `json:"road"`
			Neighbourhood string `json:"neighbourhood"`
			Suburb        string `json:"suburb"`
			CityDistrict  string `json:"city_district"`
			City          string `json:"city"`
			County        string `json:"county"`
			State         string `json:"state"`
		} `json:"address"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return AddressInfo{}, err
	}

	road := strings.TrimSpace(data.Address.Road)
	if road == "" {
		road = strings.TrimSpace(data.Name)
	}

	locality := strings.TrimSpace(data.Address.City)
	if locality == "" {
		locality = strings.TrimSpace(data.Address.Suburb)
	}
	if locality == "" {
		locality = strings.TrimSpace(data.Address.Neighbourhood)
	}

	county := strings.TrimSpace(data.Address.State)
	if county == "" {
		county = strings.TrimSpace(data.Address.County)
	}

	subCounty := strings.TrimSpace(data.Address.County)
	if subCounty == "" || subCounty == county {
		subCounty = strings.TrimSpace(data.Address.CityDistrict)
	}
	if subCounty == "" {
		subCounty = locality
	}

	var formatted string
	if road != "" && locality != "" {
		formatted = fmt.Sprintf("%s, %s", road, locality)
	} else if road != "" {
		formatted = road
	} else if locality != "" && county != "" {
		formatted = fmt.Sprintf("%s, %s", locality, county)
	} else if locality != "" {
		formatted = locality
	} else {
		formatted = data.DisplayName
	}

	return AddressInfo{
		FormattedAddress: formatted,
		SubCounty:        subCounty,
		County:           county,
	}, nil
}

func (g *ReverseGeocoder) fromBigDataCloud(ctx context.Context, lat, lng float64) (AddressInfo, error) {
	url := fmt.Sprintf("https://api.bigdatacloud.net/data/reverse-geocode-client?latitude=%.6f&longitude=%.6f&localityLanguage=en", lat, lng)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return AddressInfo{}, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return AddressInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AddressInfo{}, fmt.Errorf("bigdatacloud returned status %d", resp.StatusCode)
	}

	var data struct {
		City                 string `json:"city"`
		Locality             string `json:"locality"`
		PrincipalSubdivision string `json:"principalSubdivision"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return AddressInfo{}, err
	}

	city := strings.TrimSpace(data.City)
	if city == "" {
		city = strings.TrimSpace(data.Locality)
	}
	county := strings.TrimSpace(data.PrincipalSubdivision)

	formatted := city
	if formatted != "" && county != "" {
		formatted = fmt.Sprintf("%s, %s", formatted, county)
	}

	slog.Info("Reverse Geocoding resolved BigDataCloud", "lat", lat, "lng", lng, "formatted", formatted)

	return AddressInfo{
		FormattedAddress: formatted,
		SubCounty:        city,
		County:           county,
	}, nil
}
