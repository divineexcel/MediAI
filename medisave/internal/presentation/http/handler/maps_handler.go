package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/application/dto"
	mapsclient "github.com/medisave/app/internal/infrastructure/external/maps"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type MapsHandler struct {
	mapsClient *mapsclient.Client
}

func NewMapsHandler(mapsClient *mapsclient.Client) *MapsHandler {
	return &MapsHandler{mapsClient: mapsClient}
}

// GET /api/v1/maps/nearby?lat=&lng=&type=&radius=
func (h *MapsHandler) Nearby(c *gin.Context) {
	var req dto.NearbyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid query parameters", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	radius := req.Radius
	if radius == 0 {
		radius = 5000 // 5 km default
	}

	places, err := h.mapsClient.FindNearby(req.Latitude, req.Longitude, req.Type, radius)
	if err != nil {
		response.InternalError(c, "failed to find nearby places")
		return
	}
	response.OK(c, "nearby places", places)
}

// GET /api/v1/maps/directions?origin_lat=&origin_lng=&dest_lat=&dest_lng=
func (h *MapsHandler) Directions(c *gin.Context) {
	var req dto.DirectionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid query parameters", err.Error())
		return
	}

	directionsURL := h.mapsClient.GetDirections(req.OriginLat, req.OriginLng, req.DestLat, req.DestLng)
	response.OK(c, "directions", gin.H{"url": directionsURL})
}
