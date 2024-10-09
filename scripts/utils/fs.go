package utils

import (
	"encoding/json"
	"os"

)

// read ../recommended.json and unmarshal it into a Recommended struct
func ReadRecommended() (Recommended, error) {
	data, err := os.ReadFile("../recommended.json")
	if err != nil {
		return Recommended{}, err
	}

	var recommended Recommended
	if err := json.Unmarshal(data, &recommended); err != nil {
		return Recommended{}, err
	}

	return recommended, nil
}
