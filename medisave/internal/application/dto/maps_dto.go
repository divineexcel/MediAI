package dto

type NearbyRequest struct {
	Latitude  float64 `form:"lat"      validate:"required"`
	Longitude float64 `form:"lng"      validate:"required"`
	Type      string  `form:"type"     validate:"required,oneof=hospital clinic pharmacy laboratory blood_bank"`
	Radius    int     `form:"radius"   validate:"omitempty,min=500,max=50000"`
}

type DirectionsRequest struct {
	OriginLat    float64 `form:"origin_lat"  validate:"required"`
	OriginLng    float64 `form:"origin_lng"  validate:"required"`
	DestLat      float64 `form:"dest_lat"    validate:"required"`
	DestLng      float64 `form:"dest_lng"    validate:"required"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type NearbyPlaceResponse struct {
	PlaceID      string  `json:"place_id"`
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Distance     string  `json:"distance"`
	Phone        string  `json:"phone"`
	OpenNow      bool    `json:"open_now"`
	OpeningHours string  `json:"opening_hours"`
	Rating       float64 `json:"rating"`
	Type         string  `json:"type"`
}
