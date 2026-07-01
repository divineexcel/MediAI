package maps

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/medisave/app/internal/application/dto"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// FindNearby returns health facilities near the given coordinates.
// Falls back to curated Nigerian hospital data if no API key is configured.
func (c *Client) FindNearby(lat, lng float64, placeType string, radiusMeters int) ([]*dto.NearbyPlaceResponse, error) {
	if c.apiKey == "" {
		return localFallback(lat, lng, placeType, radiusMeters), nil
	}
	return c.googlePlaces(lat, lng, placeType, radiusMeters)
}

// GetDirections returns a simple directions URL (deep link to Google Maps).
func (c *Client) GetDirections(originLat, originLng, destLat, destLng float64) string {
	return fmt.Sprintf(
		"https://www.google.com/maps/dir/?api=1&origin=%f,%f&destination=%f,%f&travelmode=driving",
		originLat, originLng, destLat, destLng,
	)
}

// ─── GOOGLE PLACES API ───────────────────────────────────────────────────────

type googlePlacesResponse struct {
	Status  string `json:"status"`
	Results []struct {
		PlaceID  string `json:"place_id"`
		Name     string `json:"name"`
		Vicinity string `json:"vicinity"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		Rating       float64 `json:"rating"`
		OpeningHours struct {
			OpenNow bool `json:"open_now"`
		} `json:"opening_hours"`
	} `json:"results"`
}

func (c *Client) googlePlaces(lat, lng float64, placeType string, radiusMeters int) ([]*dto.NearbyPlaceResponse, error) {
	keyword := googleKeyword(placeType)
	params := url.Values{}
	params.Set("location", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("radius", fmt.Sprintf("%d", radiusMeters))
	params.Set("keyword", keyword)
	params.Set("key", c.apiKey)

	reqURL := "https://maps.googleapis.com/maps/api/place/nearbysearch/json?" + params.Encode()
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return localFallback(lat, lng, placeType, radiusMeters), nil
	}
	defer resp.Body.Close()

	var result googlePlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return localFallback(lat, lng, placeType, radiusMeters), nil
	}

	// Fall back to local data if the API key is invalid/missing or returned no results
	if result.Status != "OK" || len(result.Results) == 0 {
		return localFallback(lat, lng, placeType, radiusMeters), nil
	}

	places := make([]*dto.NearbyPlaceResponse, 0, len(result.Results))
	for _, r := range result.Results {
		dist := haversineKm(lat, lng, r.Geometry.Location.Lat, r.Geometry.Location.Lng)
		places = append(places, &dto.NearbyPlaceResponse{
			PlaceID:   r.PlaceID,
			Name:      r.Name,
			Address:   r.Vicinity,
			Latitude:  r.Geometry.Location.Lat,
			Longitude: r.Geometry.Location.Lng,
			Distance:  fmt.Sprintf("%.1f km", dist),
			OpenNow:   r.OpeningHours.OpenNow,
			Rating:    r.Rating,
			Type:      placeType,
		})
	}
	return places, nil
}

func googleKeyword(placeType string) string {
	switch placeType {
	case "hospital":
		return "hospital"
	case "clinic":
		return "clinic medical"
	case "pharmacy":
		return "pharmacy chemist"
	case "laboratory":
		return "medical laboratory"
	case "blood_bank":
		return "blood bank"
	default:
		return "hospital"
	}
}

// ─── LOCAL FALLBACK DATA ─────────────────────────────────────────────────────

type facility struct {
	name    string
	address string
	phone   string
	lat     float64
	lng     float64
	t       string
	hours   string
}

var nigerianFacilities = []facility{
	// Lagos
	{"Lagos University Teaching Hospital (LUTH)", "Idi-Araba, Surulere, Lagos", "08023032003", 6.5108, 3.3598, "hospital", "24 hours"},
	{"Lagos Island General Hospital", "1 Lagos Island, Lagos", "01-2700800", 6.4522, 3.4007, "hospital", "24 hours"},
	{"Reddington Hospital", "12 Idowu Martins, Victoria Island, Lagos", "01-2716800", 6.4281, 3.4219, "hospital", "24 hours"},
	{"St. Nicholas Hospital", "57 Campbell St, Lagos Island", "01-2700911", 6.4541, 3.3913, "hospital", "24 hours"},
	{"Eko Hospital", "31 Mobolaji Bank Anthony Way, Maryland, Lagos", "01-7900400", 6.5622, 3.3571, "hospital", "24 hours"},
	{"Mediplan Healthcare", "45A Awolowo Road, Ikoyi, Lagos", "01-4617272", 6.4476, 3.4343, "clinic", "Mon-Sat 8am-8pm"},
	{"MedPlus Pharmacy Victoria Island", "Adeola Odeku, Victoria Island, Lagos", "08180000000", 6.4298, 3.4206, "pharmacy", "8am-10pm"},
	{"Alpha Medical Laboratory", "21 Adeniran Ogunsanya, Surulere, Lagos", "08035000000", 6.5041, 3.3573, "laboratory", "7am-7pm"},
	// Abuja
	{"National Hospital Abuja", "Plot 132 Central Business District, Abuja", "09-5238101", 9.0579, 7.4951, "hospital", "24 hours"},
	{"Garki Hospital", "Garki Area 3, Abuja", "09-2340005", 9.0393, 7.4738, "hospital", "24 hours"},
	{"Nisa Premier Hospital", "Jabi, Abuja", "09-2917600", 9.0624, 7.4490, "hospital", "24 hours"},
	{"Asokoro District Hospital", "Asokoro, Abuja", "09-3140000", 9.0523, 7.5284, "hospital", "24 hours"},
	{"Wuse General Hospital", "Wuse Zone 6, Abuja", "09-5230032", 9.0676, 7.4850, "hospital", "24 hours"},
	// Kano
	{"Aminu Kano Teaching Hospital", "Zoo Road, Kano", "064-666601", 12.0022, 8.5159, "hospital", "24 hours"},
	{"Murtala Muhammad Specialist Hospital", "Kano State, Kano", "064-637300", 12.0007, 8.5182, "hospital", "24 hours"},
	// Port Harcourt
	{"University of Port Harcourt Teaching Hospital", "East-West Road, Choba, Rivers State", "084-234730", 4.8396, 6.9111, "hospital", "24 hours"},
	{"Braithwaite Memorial Specialist Hospital", "Moscow Road, Port Harcourt", "084-232611", 4.7774, 6.9978, "hospital", "24 hours"},
}

func localFallback(lat, lng float64, placeType string, _ int) []*dto.NearbyPlaceResponse {
	type distFac struct {
		f    facility
		dist float64
	}
	var all []distFac
	for _, f := range nigerianFacilities {
		if placeType != "" && f.t != placeType {
			continue
		}
		all = append(all, distFac{f, haversineKm(lat, lng, f.lat, f.lng)})
	}
	// Insertion sort ascending by distance
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].dist < all[j-1].dist; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	n := 10
	if len(all) < n {
		n = len(all)
	}
	results := make([]*dto.NearbyPlaceResponse, 0, n)
	for _, a := range all[:n] {
		results = append(results, &dto.NearbyPlaceResponse{
			PlaceID:      fmt.Sprintf("local_%s", a.f.name),
			Name:         a.f.name,
			Address:      a.f.address,
			Latitude:     a.f.lat,
			Longitude:    a.f.lng,
			Distance:     fmt.Sprintf("%.1f km", a.dist),
			Phone:        a.f.phone,
			OpenNow:      true,
			OpeningHours: a.f.hours,
			Rating:       4.0,
			Type:         a.f.t,
		})
	}
	return results
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
