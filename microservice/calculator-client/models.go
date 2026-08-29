package main

type CalcRequest struct {
	Nums []float64 `json:"nums,omitempty"`
	A    float64   `json:"a,omitempty"`
}

type CalcResponse struct {
	Result float64   `json:"result,omitempty"`
	Results []float64 `json:"results,omitempty"`
}