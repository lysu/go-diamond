package diamond_test

import (
	"testing"
	"github.com/lysu/go-diamond"
	"time"
)

func TestGetDiamondConfigWithAPI(t *testing.T) {

	diamondManager, err := diamond.NewDiamondManager("testgroup", "testdata")
	time.Sleep(10*time.Second)
	if err != nil {
		t.FailNow()
	}

	cfg, err := diamondManager.AvailableConfigureInformation(10 * time.Second)
	if err != nil {
		t.FailNow()
	}

	if "" == cfg {
		t.Fatalf("fetch cfg failure")
	}

}
