# Hotwire + Templ Integration Guide

This project has been refactored to use [Go templ](https://templ.guide/) templates and [Hotwire Turbo](https://turbo.hotwired.dev/) for a modern, mobile-friendly user interface that works great with Hotwire Native for iOS and Android apps.

## What Changed

### Architecture
- **Frontend**: Moved from JSON API to HTML-first with templ templates
- **Mobile-Friendly**: Responsive design with Tailwind CSS
- **Progressive Enhancement**: Still supports JSON responses for API clients
- **Hotwire Turbo**: Fast navigation without full page reloads
- **Hotwire Native Ready**: Works seamlessly with Hotwire Turbo Native for iOS/Android

### Project Structure
```
├── templates/
│   ├── layouts/        # Base HTML layouts
│   │   └── base.templ  # Main layout with navigation
│   ├── pages/          # Full page templates
│   │   ├── dashboard.templ
│   │   ├── worklogs.templ
│   │   └── issues.templ
│   └── components/     # Reusable UI components
│       ├── worklog.templ
│       └── issue.templ
├── static/
│   ├── css/            # Custom stylesheets
│   └── js/             # JavaScript files (if needed)
└── cmd/jiraworklog/
    ├── handlers.go     # HTTP handlers with HTML/JSON support
    └── main.go         # Application entry point
```

## Development Setup

### Prerequisites
- Go 1.25+
- [templ CLI tool](https://templ.guide/)

### Installing templ

The templ tool is already configured in `go.mod` as a tool dependency:

```bash
# Install templ CLI
go install github.com/a-h/templ/cmd/templ@latest
```

### Building the Application

1. **Generate Go code from templ files** (required after any .templ file changes):
```bash
templ generate
```

2. **Build the application**:
```bash
cd cmd/jiraworklog
go build -o ../../jiraworklog
```

Or combine both steps:
```bash
templ generate && cd cmd/jiraworklog && go build -o ../../jiraworklog
```

### Running the Application

```bash
./jiraworklog -v
```

The server will start on `http://localhost:8180` (default port).

## Features

### Content Negotiation
The application automatically serves the appropriate content type:
- **HTML**: When accessed via a web browser (default)
- **JSON**: When the `Accept: application/json` header is present

Example:
```bash
# Get HTML response
curl http://localhost:8180/worklogs

# Get JSON response
curl -H "Accept: application/json" http://localhost:8180/worklogs
```

### Mobile-Responsive Design
- Desktop: Full tables with all information
- Mobile: Card-based layout for better readability
- Touch-friendly interface
- Optimized for small screens

### Hotwire Turbo Benefits
1. **Fast Navigation**: Page transitions without full reloads
2. **Progressive Enhancement**: Works without JavaScript, better with it
3. **Mobile App Ready**: Drop-in compatibility with Turbo Native
4. **Reduced Bandwidth**: Only fetches changed content

## Hotwire Native Integration

This backend is now ready to be used with Hotwire Turbo Native for iOS and Android:

### iOS Setup
1. Create a new iOS project with Turbo Native
2. Point the initial URL to your server: `http://your-server:8180`
3. Turbo will handle all navigation automatically

### Android Setup
1. Create a new Android project with Turbo Native
2. Configure the server URL in your app
3. Navigation and page rendering work out of the box

### Example iOS Configuration
```swift
import Turbo

let session = Session()
let url = URL(string: "http://localhost:8180")!
session.visit(Visit(url: url, action: .advance))
```

## Available Routes

### Dashboard & Navigation
- `GET /` - Dashboard (home page)
- `GET /dashboard` - Dashboard

### Worklogs
- `GET /worklogs` - All worklogs
- `GET /worklogs/groupby?group={type|priority|project|weekday}&start={YYYYMMDD}&stop={YYYYMMDD}` - Grouped worklogs
- `GET /worklogs/perdev?start={YYYYMMDD}&stop={YYYYMMDD}` - Worklogs per developer
- `GET /worklogs/perdevweek` - Weekly worklogs per developer

### Issues
- `GET /issues` - All issues
- `GET /issues/groupby?group={developer|type|priority|project|status}&start={YYYYMMDD}&stop={YYYYMMDD}` - Grouped issues
- `GET /issues/accuracy?start={YYYYMMDD}&stop={YYYYMMDD}` - Issue estimation accuracy

### Static Files
- `/static/*` - CSS, JS, and other static assets

## Development Workflow

### Making Changes to Templates

1. **Edit `.templ` files** in the `templates/` directory
2. **Regenerate Go code**:
   ```bash
   templ generate
   ```
3. **Rebuild and run**:
   ```bash
   cd cmd/jiraworklog && go build -o ../../jiraworklog && cd ../.. && ./jiraworklog -v
   ```

### Hot Reload During Development

For a better development experience, use `templ generate --watch`:

Terminal 1:
```bash
templ generate --watch
```

Terminal 2:
```bash
# Use air or another hot-reload tool
go run cmd/jiraworklog/main.go -v
```

## Customization

### Styling
- The application uses [Tailwind CSS](https://tailwindcss.com/) via CDN
- Custom styles can be added to `static/css/app.css`
- For production, consider using a build step for Tailwind

### Adding New Pages

1. Create a new `.templ` file in `templates/pages/`
2. Use the base layout:
   ```go
   package pages

   import "github.com/mkobaly/jiraworklog/templates/layouts"

   templ MyNewPage(data MyDataType) {
       @layouts.Base("Page Title") {
           <h1>My New Page</h1>
           // Your content here
       }
   }
   ```
3. Add a handler in `cmd/jiraworklog/handlers.go`
4. Register the route in `cmd/jiraworklog/main.go`
5. Run `templ generate` and rebuild

### Creating Components

Reusable components go in `templates/components/`:

```go
package components

import "github.com/mkobaly/jiraworklog/types"

templ MyComponent(data *types.SomeType) {
    <div class="my-component">
        { data.Field }
    </div>
}
```

## Backward Compatibility

The old JSON API is fully preserved:
- All existing JSON endpoints continue to work
- Request with `Accept: application/json` header for JSON responses
- The `/web/` route still serves static files from the old `web/` directory

## Troubleshooting

### Templates not updating
- Make sure to run `templ generate` after changing `.templ` files
- The generated `*_templ.go` files are what Go actually compiles

### Build errors
- Ensure all imports in `.templ` files are at the top of the file
- Run `go mod tidy` to clean up dependencies
- Check that templ CLI is installed: `templ version`

### Styling issues
- Tailwind classes are loaded via CDN for simplicity
- For production, use a proper Tailwind build process
- Check browser console for any CSS loading errors

## Production Deployment

For production:

1. **Build the application**:
   ```bash
   templ generate
   cd cmd/jiraworklog && go build -o ../../jiraworklog
   ```

2. **Consider adding a Tailwind build step** for smaller CSS bundles

3. **Enable HTTPS** for Hotwire Native apps

4. **Set appropriate CORS headers** if needed for API access

## Resources

- [Templ Documentation](https://templ.guide/)
- [Hotwire Turbo](https://turbo.hotwired.dev/)
- [Hotwire Native iOS](https://github.com/hotwired/turbo-ios)
- [Hotwire Native Android](https://github.com/hotwired/turbo-android)
- [Tailwind CSS](https://tailwindcss.com/)
