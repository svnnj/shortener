package model

type ShortenBatchReq struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortenBatchRes struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ShortenJSONReq struct {
	URL string `json:"url"`
}

type ShortenJSONRes struct {
	Result string `json:"result"`
}
