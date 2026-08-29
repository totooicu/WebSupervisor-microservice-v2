package models

type AddRequest struct {
	Nums []float64 `json:"nums"`
}

type MultiplyRequest struct {
	Nums []float64 `json:"nums"`
}

type SubtractRequest struct {
	A    float64   `json:"a"`
	Nums []float64 `json:"nums"`
}

type DivideRequest struct {
	A    float64   `json:"a"`
	Nums []float64 `json:"nums"`
}

type CalcArrayResponse struct {
	Result []float64 `json:"result"`
}

type CalcSingleResponse struct {
	Result float64 `json:"result"`
}