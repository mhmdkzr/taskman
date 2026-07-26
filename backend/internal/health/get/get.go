package get

// Response is the health check result.
type Response struct {
	Status string `json:"status"`
}

func get() Response {
	return Response{Status: "ok"}
}
