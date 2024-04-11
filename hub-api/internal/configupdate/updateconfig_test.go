package configupdate

import (
	"OpcUaTimeSeriesHub/hub-api/util"

	"fmt"
	"testing"
)

func TestYourFunction(t *testing.T) {
	a := CreateConfig(util.LoadConfig().TelegrafConfigPath)
	fmt.Println(a)
}
