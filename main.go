package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/OldEphraim/gator_blog_aggregator/internal/commands"
	"github.com/OldEphraim/gator_blog_aggregator/internal/config"
	"github.com/OldEphraim/gator_blog_aggregator/internal/database"
	"github.com/OldEphraim/gator_blog_aggregator/internal/state"
)

func main() {
	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Load the database URL from the config
	dbURL := cfg.DatabaseURL

	// Open a connection to the database
	db, dbErr := sql.Open("postgres", dbURL)
	if dbErr != nil {
		log.Fatalf("Failed to connect to database: %v", dbErr)
	}
	defer db.Close() // Ensure the database connection is closed when main returns

	// Initialize the database queries struct
	dbQueries := database.New(db)

	// Initialize the state with the config and database queries
	appState := &state.State{
		Cfg: &cfg,
		DB:  dbQueries,
	}

	// Initialize the commands struct and register commands
	cmds := &commands.Commands{}
	cmds.Register("login", commands.HandlerLogin)
	cmds.Register("register", commands.HandlerRegister)
	cmds.Register("reset", commands.HandlerReset)
	cmds.Register("users", commands.HandlerUsers)
	cmds.Register("agg", commands.HandlerAgg)
	cmds.Register("addfeed", commands.MiddlewareLoggedIn(commands.HandlerAddFeed))
	cmds.Register("feeds", commands.HandlerListFeeds)
	cmds.Register("follow", commands.MiddlewareLoggedIn(commands.HandlerFollow))
	cmds.Register("following", commands.MiddlewareLoggedIn(commands.HandlerFollowing))
	cmds.Register("unfollow", commands.MiddlewareLoggedIn(commands.HandlerUnfollow))
	cmds.Register("browse", commands.HandlerBrowse)

	// Check for command-line arguments
	if len(os.Args) < 2 {
		log.Fatalf("Not enough arguments provided")
	}

	// Extract the command name and arguments
	cmd := commands.Command{
		Name: os.Args[1],
		Args: os.Args[2:], // Arguments start from the third element
	}

	// Run the command
	if err := cmds.Run(appState, cmd); err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1) // Exit with code 1 if there's an error
	}
}
