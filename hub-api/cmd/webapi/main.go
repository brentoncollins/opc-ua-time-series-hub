package main

import (
	"OpcUaTimeSeriesHub/hub-api/internal/configupdate"
	"OpcUaTimeSeriesHub/hub-api/internal/database"
	"OpcUaTimeSeriesHub/hub-api/internal/webapi"
	"OpcUaTimeSeriesHub/hub-api/util"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

func main() {

	a := configupdate.CreateConfig(util.LoadConfig().TelegrafConfigPath)
	fmt.Println(a)

	time.Sleep(15 * time.Second) // Need this to start after the database. This is a hack. Should be fixed in the future.
	r := mux.NewRouter()
	_, err := database.InitDB(false)
	if err != nil {
		return
	}
	err = database.UpdateHierarchy()
	if err != nil {
		return
	}
	// Create a ticker that fires every 30 minutes
	//ticker := time.NewTicker(30 * time.Minute)
	//go func() {
	//	for {
	//		select {
	//		case <-ticker.C:
	//			err := database.UpdateHierarchy()
	//			if err != nil {
	//				log.Fatalf("Failed to update hierarchy: %v", err)
	//			}
	//		}
	//	}
	//}()

	// Register the handlers
	r.HandleFunc("/api/nodes", webapi.GetNodesHandler).Methods("GET")
	r.HandleFunc("/api/updated-required", webapi.GetUpdatesRequired).Methods("GET")
	r.HandleFunc("/api/update-node-history", webapi.UpdateNodeHistoryHandler).Methods("POST")
	r.HandleFunc("/api/update-telegraf-config", webapi.UpdateConfigFileWithHistoryNodes).Methods("POST")

	// Start the server
	log.Println("Starting server on :9090")
	log.Fatal(http.ListenAndServe(":9090", r))
}
