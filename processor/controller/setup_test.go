package controller

import (
	"github.com/multivactech/MultiVAC/configs/config"
)

func createMockedRootController(isStorageNode bool) *RootController {
	cfg := &config.Config{
		StorageNode: isStorageNode,
	}
	return &RootController{
		cfg: cfg,
	}
}
