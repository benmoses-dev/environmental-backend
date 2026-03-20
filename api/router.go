package api

import (
	"net/http"
)

func SetupRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/temperature", TemperatureHandler)
	mux.HandleFunc("/humidity", HumidityHandler)
	mux.HandleFunc("/pressure", PressureHandler)
	return mux
}
