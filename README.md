Gator Blog Aggregator

Gator Blog Aggregator is a command-line tool designed to scrape and aggregate RSS feed data from multiple blogs. It allows users to manage feeds, create accounts, and fetch posts from different blogs, storing them in a database for future retrieval. The tool supports login and registration functionality, user-specific feed management, and feed scraping at configurable intervals.

Features

User Management: Register and log in as a user to personalize your feed scraping experience.

Feed Aggregation: Fetches RSS feed data from various blogs and stores the results in a database.

Feed Management: Add, list, and view feeds. Each user can manage their own feed subscriptions.

Periodic Scraping: Automatically scrape RSS feeds at configurable time intervals.

Feed Tracking: Tracks when feeds were last fetched to avoid duplicate scraping.

Installation
1. Clone the repository: git clone https://github.com/OldEphraim/gator_blog_aggregator.git
2. Navigate into the project directory: cd gator_blog_aggregator
3. Install dependencies (if applicable, such as any Go modules or external libraries): go mod tidy
4. Set up your database. This project relies on PostgreSQL (or similar). You’ll need to configure the database in the project’s configuration files.
5. Build the project: go build

Usage
Once everything is set up, you can run the tool via the command line. Here are the available commands:

1. Register a new user: ./gator_blog_aggregator register <username>
2. Log in as an existing user: ./gator_blog_aggregator login <username>
3. Add a new RSS feed: ./gator_blog_aggregator add-feed <feed_name> <feed_url>
4. List all available feeds: ./gator_blog_aggregator list-feeds
5. Manually scrape feeds: ./gator_blog_aggregator agg <time_between_requests>
6. List all users: ./gator_blog_aggregator users
7. Reset all users: ./gator_blog_aggregator reset

Configuration
All configurations are managed through the internal state.State and config files. You can customize the database settings, scraping intervals, and other configurations by modifying the appropriate config files or passing arguments to the commands.

Contributing
Contributions are welcome! If you'd like to contribute to this project, please open a pull request or submit an issue on GitHub.

License
This project is licensed under the MIT License.
