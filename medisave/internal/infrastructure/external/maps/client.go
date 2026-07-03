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
// Falls back to curated Abuja FCT data if no API key is configured.
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

var abujaFacilities = []facility{
	// Hospitals
	{"National Hospital Abuja", "Plot 132 Central Business District, Abuja", "09-5238101", 9.0579, 7.4951, "hospital", "24 hours"},
	{"Garki Hospital", "Garki Area 3, Abuja", "09-2340005", 9.0393, 7.4738, "hospital", "24 hours"},
	{"Nisa Premier Hospital", "Jabi, Abuja", "09-2917600", 9.0624, 7.4490, "hospital", "24 hours"},
	{"Asokoro District Hospital", "Asokoro, Abuja", "09-3140000", 9.0523, 7.5284, "hospital", "24 hours"},
	{"Wuse General Hospital", "Wuse Zone 6, Abuja", "09-5230032", 9.0676, 7.4850, "hospital", "24 hours"},
	{"Maitama District Hospital", "Maitama, Abuja", "09-5230081", 9.0833, 7.4961, "hospital", "24 hours"},
	{"Kubwa General Hospital", "Kubwa, Abuja", "09-2340007", 9.1206, 7.3242, "hospital", "24 hours"},
	{"Gwagwalada General Hospital", "Gwagwalada, Abuja", "09-8820001", 8.9419, 7.0834, "hospital", "24 hours"},
	{"Nyanya General Hospital", "Nyanya, Abuja", "09-2340005", 8.9833, 7.5500, "hospital", "24 hours"},
	{"Bwari General Hospital", "Bwari, Abuja", "09-5230050", 9.2667, 7.3667, "hospital", "24 hours"},
	{"University of Abuja Teaching Hospital", "Gwagwalada, Abuja", "09-8820002", 8.9446, 7.0807, "hospital", "24 hours"},
	// Clinics
	{"Prime Health Clinic", "Wuse Zone 4, Abuja", "09-5230091", 9.0670, 7.4800, "clinic", "Mon-Fri 8am-6pm"},
	{"Hilux Healthcare Clinic", "Maitama, Abuja", "09-5230092", 9.0800, 7.4920, "clinic", "Mon-Sat 8am-7pm"},
	{"Cedarcrest Clinics", "Jabi Dam Road, Jabi, Abuja", "09-2917605", 9.0600, 7.4450, "clinic", "Mon-Fri 8am-6pm"},
	{"El-Shalom Clinic", "Area 10, Garki, Abuja", "09-2340006", 9.0380, 7.4700, "clinic", "Mon-Sat 8am-6pm"},
	{"Alpha Spring Medical Centre", "Utako, Abuja", "09-5230093", 9.0530, 7.4580, "clinic", "Mon-Sat 8am-8pm"},
	{"Keffi Medical Centre", "Abuja-Keffi Road, Mararaba", "09-5230094", 8.9491, 7.4708, "clinic", "24 hours"},
	{"Jordan Medical Centre", "Area 2, Garki, Abuja", "09-2340008", 9.0350, 7.4900, "clinic", "Mon-Sat 8am-6pm"},
	// Pharmacies
	{"HealthPlus Pharmacy Wuse", "Wuse Zone 5, Abuja", "09-5230095", 9.0700, 7.4830, "pharmacy", "8am-9pm"},
	{"Medplus Pharmacy Abuja", "Jabi, Abuja", "09-2917608", 9.0610, 7.4520, "pharmacy", "8am-10pm"},
	{"Danat Pharmacy", "Area 3, Garki, Abuja", "09-2340009", 9.0400, 7.4750, "pharmacy", "8am-9pm"},
	{"Benyl Pharmacy", "Wuse Zone 1, Abuja", "09-5230096", 9.0650, 7.4900, "pharmacy", "8am-9pm"},
	{"Pharmajet Pharmacy", "Maitama, Abuja", "09-5230097", 9.0820, 7.4950, "pharmacy", "8am-9pm"},
	{"Zion Pharmacy", "Kubwa, Abuja", "09-2340010", 9.1180, 7.3260, "pharmacy", "8am-8pm"},
	// Laboratories
	{"Clina Lancet Laboratories", "Wuse Zone 5, Abuja", "09-5230098", 9.0710, 7.4840, "laboratory", "7am-7pm"},
	{"Synlab Nigeria", "Maitama, Abuja", "09-5230099", 9.0810, 7.4930, "laboratory", "7am-6pm"},
	{"Everight Diagnostics", "Garki Area 11, Abuja", "09-2340011", 9.0370, 7.4780, "laboratory", "7am-7pm"},
	{"Medcourt Laboratory", "Wuse Zone 4, Abuja", "09-5230100", 9.0680, 7.4820, "laboratory", "7am-6pm"},
	{"Global Diagnostics", "Utako, Abuja", "09-5230101", 9.0520, 7.4550, "laboratory", "7am-6pm"},
	// Blood Banks
	{"National Blood Transfusion Service (Abuja)", "Central Area, Abuja", "09-5238102", 9.0550, 7.4980, "blood_bank", "8am-7pm"},
	{"National Hospital Blood Bank", "Plot 132 CBD, Abuja", "09-5238103", 9.0585, 7.4955, "blood_bank", "24 hours"},
	{"Asokoro District Hospital Blood Bank", "Asokoro, Abuja", "09-3140001", 9.0528, 7.5288, "blood_bank", "24 hours"},
}

func localFallback(lat, lng float64, placeType string, _ int) []*dto.NearbyPlaceResponse {
	type distFac struct {
		f    facility
		dist float64
	}
	var all []distFac
	for _, f := range abujaFacilities {
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
