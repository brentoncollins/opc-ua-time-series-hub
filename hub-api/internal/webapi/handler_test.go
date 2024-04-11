package webapi

import (
	"OpcUaTimeSeriesHub/hub-api/internal/database"
	"OpcUaTimeSeriesHub/hub-api/util"
	"fmt"
	"testing"
)

func TestYourFunction(t *testing.T) {

	db, err := database.LoadDatabase()
	if err != nil {
		util.Logger.Error("Failed to initialize DB", err)
		return
	}
	hierarchy, _ := database.LoadHierarchy(db)

	fmt.Println(hierarchy)

}
