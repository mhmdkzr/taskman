package get

import "github.com/mhmdkzr/app/pkg/githash"

// Response is the health check result.
type Response struct {
	Status string `json:"status"`
	Commit string `json:"commit,omitempty"`
}

func get() Response {
	return Response{Status: "ok", Commit: githash.GetCommitHash()}
}
