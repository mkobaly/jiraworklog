# Quick Start Guide

## Your app is ready to use! 🎉

### Run the application

```bash
./jiraworklog -v
```

Then open your browser to: **http://localhost:8180**

## What You'll See

- **Dashboard** - Quick navigation to different sections
- **Worklogs** - All time tracking entries
  - Mobile: Card-based view
  - Desktop: Full table view
- **Issues** - All tracked Jira issues
  - Mobile-friendly cards
  - Desktop table with sorting

## Development

### After changing templates

```bash
# 1. Generate Go code from templates
templ generate

# 2. Rebuild
cd cmd/jiraworklog && go build -o ../../jiraworklog

# 3. Run
cd ../.. && ./jiraworklog -v
```

### Or use watch mode (recommended)

Terminal 1:
```bash
templ generate --watch
```

Terminal 2:
```bash
go run cmd/jiraworklog/main.go -v
```

## Testing Different Devices

### Mobile view
Resize your browser or use Chrome DevTools device emulation

### JSON API (backward compatible)
```bash
curl -H "Accept: application/json" http://localhost:8180/worklogs
```

## Building a Mobile App

Your server is now ready for Hotwire Turbo Native!

### iOS (Swift)
```swift
import Turbo

let url = URL(string: "http://localhost:8180")!
session.visit(Visit(url: url, action: .advance))
```

### Android (Kotlin)
```kotlin
session.visit(
    TurboVisit(
        location = "http://localhost:8180",
        action = TurboVisitAction.REPLACE
    )
)
```

## Features

✅ Fast page navigation (Hotwire Turbo)
✅ Mobile-responsive design
✅ Works offline with Turbo Native
✅ Progressive enhancement
✅ Backward compatible JSON API

## Documentation

- [MIGRATION_SUMMARY.md](./MIGRATION_SUMMARY.md) - What changed
- [HOTWIRE_SETUP.md](./HOTWIRE_SETUP.md) - Detailed guide

## Need Help?

Check the documentation or visit:
- https://templ.guide/ (Templates)
- https://turbo.hotwired.dev/ (Turbo)
- https://github.com/hotwired/turbo-ios (iOS)
- https://github.com/hotwired/turbo-android (Android)

Enjoy! 🚀
