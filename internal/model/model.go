package model

type DataRequest struct {
	Key string `json:"key"`
	Value string `json:"value"`
}

type Stats struct {
	Requests int64 `json:"requests"`
	Keys int `json:"keys"`
	UptimeSeconds int64 `json:"uptime_seconds"`
}