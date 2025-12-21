package gdl90

// NACpFromHorizontalAccuracyMeters maps an estimated 95% horizontal position
// accuracy (meters) to a NACp category.
//
// This mirrors Stratux's calculateNACp() behavior for interoperability.
func NACpFromHorizontalAccuracyMeters(accuracyM float64) byte {
	if accuracyM <= 0 {
		return 0
	}
	if accuracyM < 3 {
		return 11
	}
	if accuracyM < 10 {
		return 10
	}
	if accuracyM < 30 {
		return 9
	}
	if accuracyM < 92.6 {
		return 8
	}
	if accuracyM < 185.2 {
		return 7
	}
	if accuracyM < 555.6 {
		return 6
	}
	return 0
}
