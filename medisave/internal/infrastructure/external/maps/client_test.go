package maps

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func distKm(s string) float64 {
	var d float64
	fmt.Sscanf(s, "%f km", &d)
	return d
}

func TestHaversineKm(t *testing.T) {
	// Query point in Surulere → LUTH should be ~2.62 km
	assert.InDelta(t, 2.62, haversineKm(6.5244, 3.3792, 6.5108, 3.3598), 0.2)

	// Same point → zero
	assert.InDelta(t, 0.0, haversineKm(6.5244, 3.3792, 6.5244, 3.3792), 0.001)

	// Abuja→Lagos ≈ 500 km sanity check
	d := haversineKm(9.07, 7.49, 6.52, 3.38)
	assert.True(t, d > 450 && d < 600, "expected 450–600 km, got %.1f", d)
}

func TestHaversineSymmetry(t *testing.T) {
	d1 := haversineKm(6.5244, 3.3792, 9.0579, 7.4951)
	d2 := haversineKm(9.0579, 7.4951, 6.5244, 3.3792)
	assert.InDelta(t, 0.0, math.Abs(d1-d2), 0.001)
}

func TestLocalFallback_Hospital_Lagos(t *testing.T) {
	results := localFallback(6.5244, 3.3792, "hospital", 5000)

	require.NotNil(t, results)
	require.NotEmpty(t, results, "must return hospitals near Lagos")
	assert.LessOrEqual(t, len(results), 10)

	for _, r := range results {
		assert.Equal(t, "hospital", r.Type)
		assert.NotEmpty(t, r.Name)
		assert.NotEmpty(t, r.PlaceID)
		assert.True(t, r.OpenNow)
	}

	// National Hospital Abuja should appear (as it is in the curated fallback list)
	var foundNational bool
	for _, r := range results {
		if r.Name == "National Hospital Abuja" {
			foundNational = true
		}
	}
	assert.True(t, foundNational, "National Hospital Abuja must be in results")

	// Results must be sorted ascending by distance
	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, distKm(results[i-1].Distance), distKm(results[i].Distance)+0.01,
			"results[%d] distance should be >= results[%d]", i, i-1)
	}
}

func TestLocalFallback_Abuja(t *testing.T) {
	results := localFallback(9.0579, 7.4951, "hospital", 5000)
	require.NotEmpty(t, results)
	assert.Equal(t, "hospital", results[0].Type)

	// National Hospital Abuja is at the query point — should be first
	closestDist := haversineKm(9.0579, 7.4951, results[0].Latitude, results[0].Longitude)
	assert.Less(t, closestDist, 10.0, "closest Abuja hospital should be within 10 km")
}

func TestLocalFallback_BloodBank(t *testing.T) {
	// There are blood banks in curated data
	results := localFallback(6.5244, 3.3792, "blood_bank", 5000)
	require.NotNil(t, results, "must return non-nil slice so JSON marshals as [] not null")
	assert.NotEmpty(t, results)
}

func TestLocalFallback_AllTypes(t *testing.T) {
	results := localFallback(6.5244, 3.3792, "", 5000)
	require.NotEmpty(t, results)

	types := map[string]bool{}
	for _, r := range results {
		types[r.Type] = true
	}
	assert.True(t, types["hospital"])
}

func TestLocalFallback_SmallRadius_StillReturnsResults(t *testing.T) {
	// radius=1m → still returns nearest facilities (no radius filter in new impl)
	results := localFallback(6.5244, 3.3792, "hospital", 1)
	require.NotNil(t, results)
	require.NotEmpty(t, results, "should always return results regardless of radius in fallback mode")
}
