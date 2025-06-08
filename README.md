# Supreme Garbanzo: Article Summarizer Backend

This project provides a backend service for collecting article URLs from users, summarizing them using OpenAI, and sending the summaries to the user's email. It is written in Go and uses Gmail SMTP for email delivery.

**Human-note** This is an excercise in vibe-coding and should not be taken as production code. All code was AI generated and meant to be used as a learning excercise in dev tooling so would not be suprised to find some antipatterns or security loopholes lurking around.

## Features
- Accepts article URLs and user email addresses via a POST endpoint
- Stores submitted links per user
- Summarizes articles using OpenAI's API
- Sends a styled HTML email with summaries and sources to the user

## Requirements
- Go 1.18+
- An OpenAI API key
- A Gmail account with an App Password (for SMTP)

## Setup

1. **Clone the repository:**
   ```sh
   git clone <your-repo-url>
   cd supreme-garbanzo
   ```

2. **Initialize Go modules and install dependencies:**
   ```sh
   go mod tidy
   ```

3. **Set environment variables:**
   - `OPENAI_API_KEY`: Your OpenAI API key
   - `SMTP_USER`: Your Gmail address (e.g., `yourname@gmail.com`)
   - `SMTP_PASS`: Your Gmail App Password (not your regular password)

   Example:
   ```sh
   export OPENAI_API_KEY="sk-..."
   export SMTP_USER="yourname@gmail.com"
   export SMTP_PASS="your-app-password"
   ```

4. **Build and run the server:**
   ```sh
   go build -o backend/summarize-server backend/main.go
   ./backend/summarize-server
   ```
   Or use the provided script:
   ```sh
   ./scripts/start_server.sh
   ```

## API

### Submit Link
- **Endpoint:** `/submit-link`
- **Method:** `POST`
- **Body:**
  ```json
  {
    "url": "https://example.com/article",
    "email": "user@example.com"
  }
  ```
- **Response:** `200 OK` on success

## How it Works
- Each submitted link is stored per user (by email).
- When a user submits enough links (default: 5), the backend summarizes each article using OpenAI and sends a single HTML email with all summaries and sources.
- The email is styled for readability and compatibility with most email clients.

## Configuration
- Change the number of links required to trigger an email by editing `linksPerEmail` in `backend/main.go`.
- The storage directory for user links is `storage/userlinks` by default.

## Troubleshooting
- **Email not sending?** Ensure your environment variables are set and your Gmail account allows SMTP with App Passwords.
- **HTML not rendering in emails?** The backend uses only inline styles for maximum compatibility.
- **Dependency errors?** Run `go mod tidy` in the project root.

## License
MIT

## Adding the Chrome Extension to Chrome

1. **Clone this repository (if you haven't already):**
   ```sh
   git clone <your-repo-url>
   cd supreme-garbanzo
   ```

2. **Open Google Chrome and go to the Extensions page:**
   - Enter `chrome://extensions/` in the address bar and press Enter.

3. **Enable Developer Mode:**
   - Toggle the switch in the top right corner labeled "Developer mode".

4. **Load the unpacked extension:**
   - Click the "Load unpacked" button.
   - In the file dialog, navigate to the directory containing the extension source (e.g., `chrome-extension/` or the appropriate folder in this repo) and select it.

5. **The extension should now appear in your Chrome extensions list.**

6. **(Optional) Configure the extension:**
   - If the extension requires an API key or backend URL, open the extension's options or source code and update the relevant fields.

7. **Usage:**
   - Navigate to any article you want to summarize.
   - Click the extension icon in your Chrome toolbar.
   - Use the provided UI to submit the article for summarization. 