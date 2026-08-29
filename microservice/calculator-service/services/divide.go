package services

import (
	"encoding/json"
	"github.com/totooicu/go-mytool/stream"
	JsonOperate "github.com/totooicu/go-mytool/json"
	"github.com/totooicu/calculator-service/models"
)
func HandleDivide(msg *stream.StreamMessage) {
	var req models.DivideRequest
	if err := json.Unmarshal(JsonOperate.MapToJsonBytes(msg.Playload), &req); err != nil {
		stream.Response(msg, nil, 400, "invalid payload")
		return
	}
	results := make([]float64, len(req.Nums))
	for i, v := range req.Nums {
		if v == 0 {
			stream.Response(msg, nil, 500, "division by zero")
			return
		}
		results[i] = req.A / v
	}
	stream.Response(msg, map[string]interface{}{"result": results}, 0, "")
}