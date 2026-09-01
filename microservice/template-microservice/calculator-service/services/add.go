package services

import (
	"github.com/totooicu/go-mytool/stream"
	JsonOperate "github.com/totooicu/go-mytool/json"
	"github.com/totooicu/calculator-service/models"
)

func HandleAdd(msg *stream.StreamMessage) {
	var req models.AddRequest
	if err := JsonOperate.MapToJsonToStruct(msg.Playload, &req); err != nil {
		stream.ResponseErr(msg, "invalid payload")
		return
	}
	sum := 0.0
	for _, v := range req.Nums {
		sum += v
	}
	stream.ResponseSucc(msg, map[string]interface{}{"result": sum})
}