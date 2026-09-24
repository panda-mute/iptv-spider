package model

type JsonResponse[T ChannelInfo | EPGDetails] struct {
	Status  string `json:"status"`
	ErrCode string `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	Data    []T    `json:"data"`
}
