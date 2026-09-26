package models

// GetDemoExpiry safely extracts the expiry timestamp from a Demo entry
func GetDemoExpiry(val interface{}) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case map[string]interface{}:
		if exp, ok := v["expiry"]; ok {
			switch e := exp.(type) {
			case float64:
				return int64(e)
			case int64:
				return e
			}
		}
	}
	return 0
}

// SetDemo creates a normalized Demo entry
func SetDemo(expiry int64) map[string]interface{} {
	return map[string]interface{}{
		"expiry": float64(expiry),
		"warned": false,
	}
}
