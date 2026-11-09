package httpx

type Pagination struct {
	Page     int  `json:"page"`
	PageSize int  `json:"pageSize"`
	Count    int  `json:"count"`
	HasNext  bool `json:"hasNext"`
}

type Meta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Response struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Data    any      `json:"data,omitempty"`
	Error   string   `json:"error,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	Meta    *Meta    `json:"meta,omitempty"`
}
