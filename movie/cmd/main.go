package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	metadataGateway "movieservice/movie/gateway/metadata/http"
	ratingGateway "movieservice/movie/gateway/rating/http"
	"movieservice/movie/internal/controller/movie"
	httpHandler "movieservice/movie/internal/handler/http"
	"movieservice/pkg/discovery"
	"movieservice/pkg/discovery/consul"
	"net/http"
	"time"
)

const serviceName = "movie"

func main() {
	var port int
	flag.IntVar(&port, "port", 8083, "API handler port")
	flag.Parse()

	log.Printf("Starting the movie service on port %d", port)
	registry, err := consul.NewRegistry("localhost:8500")
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err = registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		panic(err)
	}

	go func() {
		for {
			if err = registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("Failed to report healthy state: " + err.Error())
			}
			time.Sleep(1 * time.Second)
		}
	}()

	defer func(registry *consul.Registry, ctx context.Context, instanceID string, _ string) {
		err = registry.Deregister(ctx, instanceID, "")
		if err != nil {
			panic(err)
		}
	}(registry, ctx, instanceID, serviceName)

	metadata := metadataGateway.New(registry)
	rating := ratingGateway.New(registry)
	svc := movie.New(rating, metadata)
	h := httpHandler.New(svc)
	http.Handle("/movie", http.HandlerFunc(h.GetMovieDetails))
	log.Printf("Listening :%d", port)
	if err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		panic(err)
	}
}
