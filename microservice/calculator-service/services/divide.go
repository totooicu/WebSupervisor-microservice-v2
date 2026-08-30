package services

import (
	"github.com/totooicu/go-mytool/stream"
	JsonOperate "github.com/totooicu/go-mytool/json"
	"github.com/totooicu/calculator-service/models"
)
func HandleDivide(msg *stream.StreamMessage) {
	var req models.DivideRequest
	if err := JsonOperate.MapToJsonToStruct(msg.Playload, &req); err != nil {
		stream.ResponseErr(msg, "invalid payload")
		return
	}
	results := make([]float64, len(req.Nums))
	for i, v := range req.Nums {
		if v == 0 {
			stream.ResponseErr(msg, "division by zero")
			return
		}
		results[i] = req.A / v
	}
	stream.ResponseSucc(msg, map[string]interface{}{"result": results})
}