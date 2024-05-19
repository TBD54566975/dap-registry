package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/tbd54566975/web5-go/dids/didweb"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("must provide domain")
	}

	domain := os.Args[1]

	registryURL, err := url.Parse(domain + "/dap-registry")
	if err != nil {
		log.Fatalf("failed to parse domain into registry service endpoint url: %v", err)
	}

	if registryURL.Scheme == "" {
		registryURL.Scheme = "https"
	}

	bearerDID, err := didweb.Create(domain, didweb.Service(
		"dap-registry",
		"dap-registry",
		registryURL.String()),
	)

	if err != nil {
		log.Fatalf("failed to generate did:web %v", err.Error())
	}

	portableDID, err := bearerDID.ToPortableDID()
	if err != nil {
		log.Fatalf("failed to export portable did: %v", err.Error())
	}

	marshaled, err := json.Marshal(portableDID)
	if err != nil {
		log.Fatalf("failed to marshal portable did: %v", err.Error())
	}

	fmt.Print(string(marshaled))
}
