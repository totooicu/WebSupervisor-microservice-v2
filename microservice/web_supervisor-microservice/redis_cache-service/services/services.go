package services

import (
	"encoding/json"
	"github.com/totooicu/redis_cache-service/models"
)
func CompareData(oldData, newData any) bool {
	if oldData == nil {
		return true
	}

	oldJSON, _ := json.Marshal(oldData)
	newJSON, _ := json.Marshal(newData)

	return string(oldJSON) != string(newJSON)
}

func SaveData(key string, data any)  error {
	return models.REDIS_CLIENT.SetKey(key, data, 0)
}

func  generateKey(app, key string) string {
	return app + ":" + key
}