package services

import (
	"encoding/json"
	"github.com/totooicu/go-mytool/stream"
	JsonOperate "github.com/totooicu/go-mytool/json"
	"github.com/totooicu/calculator-service/models"
)

func HandleAdd(msg *stream.StreamMessage) {
	var req models.AddRequest
	if err := json.Unmarshal(JsonOperate.MapToJsonBytes(msg.Playload), &req); err != nil {
		stream.Response(msg, nil, 400, "invalid payload")
		return
	}
	sum := 0.0
	for _, v := range req.Nums {
		sum += v
	}
	stream.Response(msg, map[string]interface{}{"result": sum}, 0, "")
}