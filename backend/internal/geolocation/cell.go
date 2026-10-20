package geolocation

import (
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

const defaultOpenCellIDEndpoint = "https://opencellid.org/ajax/searchCell.php"

type Coordinates struct {
	Latitude  float64
	Longitude float64
	Accuracy  float64
	Found     bool
	ExpiresAt time.Time
}

type CellResolver struct {
	client             *http.Client
	cacheMu            sync.RWMutex
	cache              map[string]Coordinates
	lastKnownValid     Coordinates
	openCellIDEndpoint string
	cacheTTL           time.Duration
}

type resolvedTower struct {
	lat     float64
	lng     float64
	acc     float64
	rssiDBm int
}

func NewCellResolver() *CellResolver {
	return &CellResolver{
		client: &http.Client{
			Timeout: 4 * time.Second,
		},
		cache:              make(map[string]Coordinates),
		openCellIDEndpoint: defaultOpenCellIDEndpoint,
		cacheTTL:           24 * time.Hour,
	}
}

func (r *CellResolver) SetEndpoint(endpoint string) {
	r.openCellIDEndpoint = endpoint
}

func (r *CellResolver) SetCacheTTL(ttl time.Duration) {
	r.cacheTTL = ttl
}

func (r *CellResolver) Resolve(ctx context.Context, mcc, mnc, lac, cid int) (float64, float64, bool) {
	lat, lng, _, ok := r.ResolveMulti(ctx, mcc, mnc, []domain.CellTower{
		{LAC: lac, CID: cid},
	})
	return lat, lng, ok
}

func (r *CellResolver) ResolveMulti(ctx context.Context, mcc, mnc int, cells []domain.CellTower) (float64, float64, float64, bool) {
	if mcc <= 0 || len(cells) == 0 {
		return 0, 0, 0, false
	}

	primary := cells[0]
	if primary.LAC <= 0 || primary.CID <= 0 {
		return 0, 0, 0, false
	}

	cacheKey := buildCellCacheKey(mcc, mnc, cells)

	if lat, lng, acc, ok := r.checkCache(cacheKey); ok {
		return lat, lng, acc, true
	}

	now := time.Now()
	resolved := make([]resolvedTower, 0, len(cells))

	for _, c := range cells {
		if c.LAC <= 0 || c.CID <= 0 {
			continue
		}

		singleKey := fmt.Sprintf("single:%d:%d:%d:%d", mcc, mnc, c.LAC, c.CID)
		if cachedLat, cachedLng, cachedAcc, ok := r.checkCache(singleKey); ok {
			resolved = append(resolved, resolvedTower{
				lat:     cachedLat,
				lng:     cachedLng,
				acc:     cachedAcc,
				rssiDBm: c.RSSIDBm,
			})
			continue
		}

		lat, lng, acc, ok := r.fromOpenCelliD(ctx, mcc, mnc, c.LAC, c.CID)
		if !ok {
			continue
		}

		r.recordResolved(singleKey, mcc, mnc, c.LAC, c.CID, lat, lng, acc, now)
		resolved = append(resolved, resolvedTower{
			lat:     lat,
			lng:     lng,
			acc:     acc,
			rssiDBm: c.RSSIDBm,
		})
	}

	if len(resolved) == 1 {
		t := resolved[0]
		r.recordResolved(cacheKey, mcc, mnc, primary.LAC, primary.CID, t.lat, t.lng, t.acc, now)
		return t.lat, t.lng, t.acc, true
	}

	if len(resolved) > 1 {
		lat, lng, acc := multilaterate(resolved)
		slog.Info("LBS Multi-cell multilaterated", "resolved_towers", len(resolved), "total_cells", len(cells), "lat", lat, "lng", lng, "accuracy", acc)
		r.recordResolved(cacheKey, mcc, mnc, primary.LAC, primary.CID, lat, lng, acc, now)
		return lat, lng, acc, true
	}

	if lat, lng, acc, ok := r.resolveFallback(mcc, mnc, primary.LAC); ok {
		return lat, lng, acc, true
	}

	r.cacheMu.Lock()
	r.cache[cacheKey] = Coordinates{Found: false, ExpiresAt: now.Add(10 * time.Minute)}
	r.cacheMu.Unlock()

	return 0, 0, 0, false
}

func multilaterate(towers []resolvedTower) (float64, float64, float64) {
	var sumLat, sumLng, sumWeight float64

	for _, t := range towers {
		weight := calculateTowerWeight(t.rssiDBm, t.acc)
		sumLat += t.lat * weight
		sumLng += t.lng * weight
		sumWeight += weight
	}

	centroidLat := sumLat / sumWeight
	centroidLng := sumLng / sumWeight

	var weightedDistanceSum float64
	minTowerAccuracy := towers[0].acc

	for _, t := range towers {
		weight := calculateTowerWeight(t.rssiDBm, t.acc)
		distMeters := haversineDistanceKM(centroidLat, centroidLng, t.lat, t.lng) * 1000.0
		weightedDistanceSum += distMeters * weight

		if t.acc > 0 && (minTowerAccuracy == 0 || t.acc < minTowerAccuracy) {
			minTowerAccuracy = t.acc
		}
	}

	geometricSpread := weightedDistanceSum / sumWeight
	accuracy := geometricSpread + (minTowerAccuracy / math.Sqrt(float64(len(towers))))

	return centroidLat, centroidLng, accuracy
}

func calculateTowerWeight(rssiDBm int, acc float64) float64 {
	if rssiDBm < 0 {
		return math.Pow(10, float64(rssiDBm)/20.0)
	}
	if acc > 0 {
		return 1.0 / acc
	}
	return 1.0
}

func (r *CellResolver) checkCache(cacheKey string) (float64, float64, float64, bool) {
	r.cacheMu.RLock()
	cached, exists := r.cache[cacheKey]
	r.cacheMu.RUnlock()

	if !exists {
		return 0, 0, 0, false
	}
	if time.Now().After(cached.ExpiresAt) {
		return 0, 0, 0, false
	}
	if !cached.Found {
		return 0, 0, 0, false
	}
	return cached.Latitude, cached.Longitude, cached.Accuracy, true
}

func (r *CellResolver) resolveFallback(mcc, mnc, lac int) (float64, float64, float64, bool) {
	lacKey := fmt.Sprintf("lac:%d:%d:%d", mcc, mnc, lac)

	r.cacheMu.RLock()
	lacCoord, hasLAC := r.cache[lacKey]
	lastValid := r.lastKnownValid
	r.cacheMu.RUnlock()

	if hasLAC && lacCoord.Found && (lacCoord.Latitude != 0 || lacCoord.Longitude != 0) {
		slog.Info("LBS fallback to last known cell in same LAC", "lac", lac, "lat", lacCoord.Latitude, "lng", lacCoord.Longitude, "accuracy", lacCoord.Accuracy)
		return lacCoord.Latitude, lacCoord.Longitude, lacCoord.Accuracy, true
	}

	if lastValid.Found && (lastValid.Latitude != 0 || lastValid.Longitude != 0) {
		slog.Info("LBS fallback to last correct location of cell found", "lat", lastValid.Latitude, "lng", lastValid.Longitude, "accuracy", lastValid.Accuracy)
		return lastValid.Latitude, lastValid.Longitude, lastValid.Accuracy, true
	}

	return 0, 0, 0, false
}

func (r *CellResolver) recordResolved(cacheKey string, mcc, mnc, lac, cid int, lat, lng, acc float64, now time.Time) {
	coord := Coordinates{
		Latitude:  lat,
		Longitude: lng,
		Accuracy:  acc,
		Found:     true,
		ExpiresAt: now.Add(r.cacheTTL),
	}

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.cache[cacheKey] = coord
	r.cache[fmt.Sprintf("single:%d:%d:%d:%d", mcc, mnc, lac, cid)] = coord
	r.cache[fmt.Sprintf("lac:%d:%d:%d", mcc, mnc, lac)] = coord
	r.lastKnownValid = coord
}

func (r *CellResolver) fromOpenCelliD(ctx context.Context, mcc, mnc, lac, cid int) (float64, float64, float64, bool) {
	if r.openCellIDEndpoint == "" {
		return 0, 0, 0, false
	}

	url := fmt.Sprintf("%s?mcc=%d&mnc=%d&lac=%d&cell_id=%d", r.openCellIDEndpoint, mcc, mnc, lac, cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, 0, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, 0, 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, false
	}

	var data struct {
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		Lng      float64 `json:"lng"`
		Range    float64 `json:"range"`
		Accuracy float64 `json:"accuracy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, 0, false
	}

	lng := data.Lon
	if lng == 0 && data.Lng != 0 {
		lng = data.Lng
	}

	if data.Lat == 0 && lng == 0 {
		return 0, 0, 0, false
	}

	accuracy := data.Range
	if accuracy <= 0 && data.Accuracy > 0 {
		accuracy = data.Accuracy
	}

	return data.Lat, lng, accuracy, true
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
