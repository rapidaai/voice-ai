package commons

type Response struct {
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type ErrorMessage struct {
	Code    int   `json:"code"`
	Message error `json:"message"`
}

type HealthCheck struct {
	Healthy bool `json:"healthy"`
}
