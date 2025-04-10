package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/snyk/snyk-code-review-exercise/api"
)

func main() {
	handler := api.New()
	fmt.Println("Server running on http://localhost:3000/")
	if err := http.ListenAndServe("localhost:3000", handler); err != nil {
		// Review Comments: Beyond Scope
		// We could start a timer everytime we walk a tree for a package
		// periodically log to the user that we are still walking

		// Review Comments:
		// We need to handle special URLs the require encoding: http://localhost:3000/package/@snyk/snyk-docker-plugin/6.15.2
		fmt.Println(err)
		os.Exit(1)
	}
}
