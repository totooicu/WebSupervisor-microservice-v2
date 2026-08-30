package services

import (
	"github.com/totooicu/go-mytool/stream"
	JsonOperate "github.com/totooicu/go-mytool/json"
	"github.com/totooicu/email-service/models"
)

func HandleEmailByConfig(msg *stream.StreamMessage) {
	var req models.EmailRequestByConfig
	if err := JsonOperate.MapToJsonToStruct(msg.Playload, &req); err != nil {
		stream.ResponseErr(msg, "invalid payload")
		return
	}
	var emailConfig *models.EmailConfig
	if eC, ok := models.EMAILS[req.EmailChoose]; !ok {
		emailConfig = models.EMAILS[models.DEFAULT_EMAIL]
	} else{
		emailConfig = eC
	}
	if emailConfig == nil {
		stream.ResponseErr(msg, "invalid email choose")
		return
	}
	if err := SendEmailByConfig(&req.EmailContent,emailConfig); err != nil {
		stream.ResponseErr(msg, err.Error())
		return
	}else{
		stream.ResponseSucc(msg, nil)
	}
}
func HandleEmailByCustom(msg *stream.StreamMessage) {
	var req models.EmailRequestByCustom
	if err := JsonOperate.MapToJsonToStruct(msg.Playload, &req); err != nil {
		stream.ResponseErr(msg, "invalid payload")
		return
	}
	if err := SendEmailByConfig(&req.EmailContent,&req.EmailConfig); err != nil {
		stream.ResponseErr(msg, err.Error())
		return
	}else{
		stream.ResponseSucc(msg, nil)
	}
}
