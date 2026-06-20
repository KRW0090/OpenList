package rakuten_drive

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	driver.RootPath
	RefreshToken string `json:"refresh_token" required:"true"`
	IDToken      string
	UID          string
}

var config = driver.Config{
	Name:              "Rakuten Drive",
	DefaultRoot:       "",
	NoUpload:          false,
	NoOverwriteUpload: true,
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &RakutenDrive{}
	})
}
