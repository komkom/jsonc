package reader

import (
	"os"

	log "github.com/Sirupsen/logrus"
)

func init() {

	//log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
}
