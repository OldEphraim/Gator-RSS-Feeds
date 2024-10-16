package state

import (
	"github.com/OldEphraim/gator_blog_aggregator/internal/config"
	"github.com/OldEphraim/gator_blog_aggregator/internal/database"
)

type State struct {
	Cfg *config.Config
	DB  *database.Queries
}
