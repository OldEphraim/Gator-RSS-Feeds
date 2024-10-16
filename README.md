# Gator-RSS-Feeds

A simple RSS feed aggregator written in Rust.

## Features
- Fetch RSS feeds from multiple sources
- Parse and display feed content
- Cache feed data for offline use
- Command-line interface (CLI)

## Installation

1. Clone the repository:

   git clone https://github.com/OldEphraim/Gator-RSS-Feeds.git  
   cd Gator-RSS-Feeds

2. Build the project:

   cargo build --release

3. Run the application:

   ./target/release/gator-rss-feeds

## Usage

Once the application is built, you can run it from the command line. For example:

   ./gator-rss-feeds --url <RSS_FEED_URL>

### Options
- `--url <RSS_FEED_URL>`: Specify an RSS feed URL to fetch and display.
- `--cache`: Cache the feed data for offline use.
- `--help`: Display the help message.

## Example

Fetching a single feed:

   ./gator-rss-feeds --url https://example.com/rss.xml

Fetching a feed and caching it:

   ./gator-rss-feeds --url https://example.com/rss.xml --cache

## Contributing

1. Fork the repository
2. Create a new branch (`git checkout -b feature-branch`)
3. Make your changes
4. Commit your changes (`git commit -m "Add feature"`)
5. Push to the branch (`git push origin feature-branch`)
6. Create a pull request

## License

This project is licensed under the MIT License.
