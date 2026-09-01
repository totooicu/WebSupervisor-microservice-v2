package services

import (
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/redis_cache-service/models"
	"github.com/sirupsen/logrus"
	"encoding/json"


)
	// stream.RegisterService("compare_and_save", services.handleCompareAndSave)
	// stream.RegisterService("get", services.handleGet)
	// stream.RegisterService("set", services.handleSet)
	// stream.RegisterService("delete", services.handleDelete)
	// stream.RegisterService("get_and_set", services.handleGetAndSet)

func  HandleCompareAndSave(msg *stream.StreamMessage) {
	// 解析参数
	var params models.CacheParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Printf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Printf("Error unmarshalling playload: %v", err)
		return
	}

	key := generateKey(params.App, params.Key)

	var oldData any
	err = models.REDIS_CLIENT.GetKey(key, &oldData)
	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf(">>>[msg.playload:%v | p.app:%s | p.key:%s | key:%v ]", msg.Playload, params.App, params.Key, key)
		logrus.Debugf("Debug - Key not found, saving new data: %v", err)
		return
	}
	paramData := map[string]interface{}{"changed": false}
		if CompareData(oldData, params.Data) {
		logrus.Printf("Data changed, saving and sending notification")

		if err := SaveData(key, params.Data); err != nil {
			stream.ResponseErr(msg, err.Error())
			logrus.Printf("Error saving data: %v", err)
			return
		}
			logrus.Debugf("Debug - Data saved successfully for key: %s", key)
	

		paramData["changed"] = true
	} else  {
		logrus.Debugf("Debug - Data unchanged for key: %s", key)
	}
	stream.ResponseSucc(msg,paramData)
	
}

func HandleGet(msg *stream.StreamMessage) {
	// 解析参数
	var params models.CacheParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error unmarshalling playload: %v", err)
		return
	}

	key := generateKey(params.App, params.Key)

	var data interface{}
	err = models.REDIS_CLIENT.GetKey(key, &data)

	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Debugf("Debug - Error getting data: %v", err)
	
	} else {
		logrus.Debugf("Debug - Got data for key: %s", key)
	}

	paramData := map[string]interface{}{
		"key":   key,
		"data":  data,
		"error": err != nil,
	}

	stream.ResponseSucc(msg,paramData)

}

func HandleSet(msg *stream.StreamMessage) {
	// 解析参数
	var params models.CacheParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error unmarshalling playload: %v", err)
		return
	}

	key := generateKey(params.App, params.Key)

		if err := SaveData(key, params.Data); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error saving data: %v", err)
		return
	}

	paramData := map[string]interface{}{
		"key":   key,
		"error": nil,
	}
	stream.ResponseSucc(msg,paramData)
}

func HandleDelete(msg *stream.StreamMessage) {
	// 解析参数
	var params models.CacheParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Printf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		stream.ResponseErr(msg, err.Error())	
		logrus.Errorf("Error unmarshalling playload: %v", err)
		return
	}

	key := generateKey(params.App, params.Key)

		if err := models.REDIS_CLIENT.DeleteKey(key); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error deleting key: %v", err)
		return
	}
	
	paramData := map[string]interface{}{
		"key":   key,
		"error": nil,
	}
	stream.ResponseSucc(msg,paramData)
	
}

func HandleGetAndSet(msg *stream.StreamMessage) {
	// 解析参数
	var params models.CacheParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error unmarshalling playload: %v", err)
		return
	}

	key := generateKey(params.App, params.Key)

	var oldData interface{}
	err = models.REDIS_CLIENT.GetKey(key, &oldData)

	if err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Debugf("Debug - Key not found, setting new data: %v", err)
		return
	} else {
		logrus.Debugf("Debug - Found existing data for key: %s", key)
	}

	if err := SaveData(key, params.Data); err != nil {
		stream.ResponseErr(msg, err.Error())
		logrus.Errorf("Error saving data: %v", err)
		return
	}

	paramData := map[string]interface{}{
		"key":      key,
		"old_data": oldData,
		"changed":   CompareData(oldData, params.Data),
		"error":    nil,
	}

	stream.ResponseSucc(msg,paramData)
}