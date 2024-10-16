package commands

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/OldEphraim/gator_blog_aggregator/internal/database"
	"github.com/OldEphraim/gator_blog_aggregator/internal/state"
	"github.com/google/uuid"
)

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	Handlers map[string]func(*state.State, Command) error
}

type RSSFeed struct {
	Title       string `xml:"channel>title"`
	Description string `xml:"channel>description"`
	Items       []Item `xml:"channel>item"`
}

type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

func (c *Commands) Register(name string, f func(*state.State, Command) error) {
	if c.Handlers == nil {
		c.Handlers = make(map[string]func(*state.State, Command) error)
	}
	c.Handlers[name] = f
}

func (c *Commands) Run(s *state.State, cmd Command) error {
	if handler, exists := c.Handlers[cmd.Name]; exists {
		return handler(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.Name)
}

func MiddlewareLoggedIn(handler func(s *state.State, cmd Command, user database.User) error) func(*state.State, Command) error {
	return func(s *state.State, cmd Command) error {
		// Fetch the user based on the current username
		user, err := s.DB.GetUser(context.Background(), s.Cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to get current user: %v", err)
		}

		// Call the handler with the user
		return handler(s, cmd, user)
	}
}

func HandlerLogin(s *state.State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("username is required")
	}

	username := cmd.Args[0]

	// Check if the user exists in the database
	existingUser, err := s.DB.GetUser(context.Background(), username)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("user %s does not exist", username)
		}
		return fmt.Errorf("error checking for existing user: %v", err)
	}

	// If the user exists, set it in the config
	s.Cfg.CurrentUserName = existingUser.Name

	// Save the updated config
	if err := s.Cfg.SetUser(existingUser.Name); err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}

	fmt.Printf("Current user set to: %s\n", existingUser.Name)
	return nil
}

func HandlerRegister(s *state.State, cmd Command) error {
	// Ensure a name was passed in the arguments
	if len(cmd.Args) == 0 {
		return fmt.Errorf("name is required")
	}

	name := cmd.Args[0]

	// Check if the user already exists
	existingUser, err := s.DB.GetUser(context.Background(), name)
	if err == nil && existingUser.Name == name {
		return fmt.Errorf("user with name %s already exists", name)
	} else if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("error checking for existing user: %v", err)
	}

	// Generate a new UUID for the user
	userID := uuid.New()

	// Set the current time for created_at and updated_at
	now := time.Now()

	// Create a new user in the database
	_, err = s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	// Update the current user in the config
	s.Cfg.CurrentUserName = name
	if err := s.Cfg.SetUser(name); err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}

	// Print success message and log user data
	fmt.Printf("User %s created successfully with ID: %s\n", name, userID)
	log.Printf("User created: %+v\n", map[string]interface{}{
		"id":         userID,
		"name":       name,
		"created_at": now,
		"updated_at": now,
	})

	return nil
}

func HandlerReset(s *state.State, cmd Command) error {
	// Call the ResetUsers query to delete all users
	if err := s.DB.ResetUsers(context.Background()); err != nil {
		return fmt.Errorf("failed to reset users: %v", err)
	}

	fmt.Println("All users have been successfully deleted.")
	return nil
}

func HandlerUsers(s *state.State, cmd Command) error {
	// Retrieve all users from the database
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to retrieve users: %v", err)
	}

	// Print the users in the specified format
	for _, user := range users {
		if user.Name == s.Cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

// HandlerAgg runs the feed scraping process every time_between_reqs duration.
func HandlerAgg(s *state.State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("time_between_reqs is required")
	}

	// Parse the time_between_reqs argument
	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	// Print the collection message
	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	// Create a ticker to run scrapeFeeds at regular intervals
	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	// Run scrapeFeeds immediately, and then every time the ticker ticks
	for {
		scrapeFeeds(s)
		<-ticker.C // Wait for the next tick
	}
}

// scrapeFeeds gets the next feed from the database, marks it as fetched, fetches its contents, and prints item titles.
func scrapeFeeds(s *state.State) {
	// Get the next feed to fetch from the database
	feed, err := s.DB.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Printf("Error fetching next feed: %v\n", err)
		return
	}

	// Mark the feed as fetched
	err = s.DB.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		fmt.Printf("Error marking feed as fetched: %v\n", err)
		return
	}

	// Fetch the feed using its URL
	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Printf("Error fetching feed %s: %v\n", feed.Url, err)
		return
	}

	// Iterate over each item in the feed
	for _, item := range rssFeed.Items {
		// Parse the publication date (optional, depends on your schema)
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			pubDate = time.Now() // Use current time if parsing fails
		}

		// Insert the feed item into the database
		_, err = s.DB.CreatePost(context.Background(), database.CreatePostParams{
			FeedID:      feed.ID,
			Title:       item.Title,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			Url:         item.Link,
			PublishedAt: sql.NullTime{Time: pubDate, Valid: !pubDate.IsZero()},
		})
		if err != nil {
			fmt.Printf("Error saving feed item: %v\n", err)
			continue
		}

		// Print success message for each saved item
		fmt.Printf("Saved Feed Item: %s\n", item.Title)
	}

	// Print the titles of feed items
	for _, item := range rssFeed.Items {
		fmt.Printf("Feed Item Title: %s\n", item.Title)
	}
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	// Create a new request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set the User-Agent header
	req.Header.Add("User-Agent", "gator")

	// Create an HTTP client and do the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching feed: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Unmarshal the XML into the RSSFeed struct
	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %v", err)
	}

	// Unescape HTML entities in Title and Description
	feed.Title = html.UnescapeString(feed.Title)
	feed.Description = html.UnescapeString(feed.Description)
	for i := range feed.Items {
		feed.Items[i].Title = html.UnescapeString(feed.Items[i].Title)
		feed.Items[i].Description = html.UnescapeString(feed.Items[i].Description)
	}

	return &feed, nil
}

func HandlerAddFeed(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("two arguments are required: name and url")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	// Create a new feed in the database
	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		Name:   name,
		Url:    url,
		UserID: uuid.NullUUID{UUID: user.ID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create feed: %v", err)
	}

	// Automatically follow the feed
	_, err = s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to follow feed after adding: %v", err)
	}

	// Print the fields of the new feed record
	fmt.Printf("Feed created successfully:\n")
	fmt.Printf("ID: %s\n", feed.ID)
	fmt.Printf("Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)

	// Check if UserID is valid and print accordingly
	if feed.UserID.Valid {
		fmt.Printf("User ID: %s\n", feed.UserID.UUID.String())
	} else {
		fmt.Printf("User ID: NULL\n")
	}

	// Check if CreatedAt is valid and print accordingly
	if feed.CreatedAt.Valid {
		fmt.Printf("Created At: %s\n", feed.CreatedAt.Time.String())
	} else {
		fmt.Printf("Created At: NULL\n")
	}

	// Check if UpdatedAt is valid and print accordingly
	if feed.UpdatedAt.Valid {
		fmt.Printf("Updated At: %s\n", feed.UpdatedAt.Time.String())
	} else {
		fmt.Printf("Updated At: NULL\n")
	}

	return nil
}

func HandlerListFeeds(s *state.State, cmd Command) error {
	// Fetch all feeds with user names
	feeds, err := s.DB.GetFeedsWithUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch feeds: %v", err)
	}

	// Check if there are any feeds
	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	// Print the feeds
	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("Name: %s\n", feed.FeedName)
		fmt.Printf("URL: %s\n", feed.Url)
		fmt.Printf("Created by: %s\n", feed.UserName)
		fmt.Println() // Add a new line for better readability
	}

	return nil
}

func HandlerFollow(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: follow <url>")
	}
	url := cmd.Args[0]

	// Look up the feed by URL
	feed, err := s.DB.GetFeedByURL(context.Background(), url)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return fmt.Errorf("failed to find feed with URL %s: %v", url, err)
	}

	// Create feed follow
	feedFollow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to follow feed: %v", err)
	}

	fmt.Printf("User %s is now following feed %s\n", feedFollow.UserName, feedFollow.FeedName)
	return nil
}

func HandlerFollowing(s *state.State, cmd Command, user database.User) error {
	feedFollows, err := s.DB.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %v", err)
	}

	if len(feedFollows) == 0 {
		fmt.Println("You are not following any feeds.")
		return nil
	}

	fmt.Println("Feeds you are following:")
	for _, ff := range feedFollows {
		fmt.Printf("- %s\n", ff.FeedName)
	}
	return nil
}

func HandlerUnfollow(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}

	url := cmd.Args[0]

	// Look up the feed by URL
	feed, err := s.DB.GetFeedByURL(context.Background(), url)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("no feed found with URL %s", url)
		}
		return fmt.Errorf("failed to look up feed with URL %s: %v", url, err)
	}

	// Delete the feed follow record
	err = s.DB.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed: %v", err)
	}

	fmt.Printf("You have unfollowed the feed: %s\n", feed.Name)
	return nil
}

// HandlerBrowse allows users to browse a specific feed and view its latest items, with an optional limit.
func HandlerBrowse(s *state.State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("usage: browse <feed_url> [limit]")
	}

	feedURL := cmd.Args[0]
	limit := 2 // default limit

	// Check if a limit was provided as the second argument
	if len(cmd.Args) > 1 {
		parsedLimit, err := strconv.Atoi(cmd.Args[1])
		if err != nil || parsedLimit <= 0 {
			return fmt.Errorf("invalid limit: must be a positive integer")
		}
		limit = parsedLimit
	}

	// Fetch the RSS feed data from the provided URL
	rssFeed, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("error fetching feed: %v", err)
	}

	// Display the feed title and items, applying the limit
	fmt.Printf("Browsing feed: %s\n", rssFeed.Title)
	for i, item := range rssFeed.Items {
		if i >= limit {
			break
		}
		fmt.Printf("  - %s\n    %s\n    Link: %s\n", item.Title, item.Description, item.Link)
	}

	return nil
}
