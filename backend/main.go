package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

type SubmitRequest struct {
	URL   string `json:"url"`
	Email string `json:"email"`
}

type UserLinks struct {
	Email string   `json:"email"`
	Links []string `json:"links"`
}

type OpenAIResponse struct {
	ID        string  `json:"id"`
	Object    string  `json:"object"`
	CreatedAt int64   `json:"created_at"`
	Status    string  `json:"status"`
	Error     *string `json:"error"`
	Model     string  `json:"model"`
	Output    []struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Status  string `json:"status"`
		Role    string `json:"role"`
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Annotations []struct {
				Type       string `json:"type"`
				StartIndex int    `json:"start_index"`
				EndIndex   int    `json:"end_index"`
				Title      string `json:"title"`
				URL        string `json:"url"`
			} `json:"annotations"`
		} `json:"content"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

const (
	storageDir    = "storage/userlinks"
	linksPerEmail = 5
)

func main() {
	os.MkdirAll(storageDir, 0755)
	http.HandleFunc("/submit-link", handleSubmitLink)
	http.HandleFunc("/queue-count", handleQueueCount)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleSubmitLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("Received request - URL: %s, Email: %s", req.URL, req.Email)

	if req.URL == "" || req.Email == "" {
		log.Printf("Missing required fields - URL: %s, Email: %s", req.URL, req.Email)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userFile := filepath.Join(storageDir, sanitizeFilename(req.Email)+".json")
	var userLinks UserLinks
	if data, err := os.ReadFile(userFile); err == nil {
		json.Unmarshal(data, &userLinks)
	} else {
		userLinks = UserLinks{Email: req.Email, Links: []string{}}
	}
	userLinks.Links = append(userLinks.Links, req.URL)
	if len(userLinks.Links) >= linksPerEmail {
		go processAndSendSummaries(userLinks)
		userLinks.Links = []string{} // Clear after sending
	}
	data, _ := json.MarshalIndent(userLinks, "", "  ")
	os.WriteFile(userFile, data, 0644)
	w.WriteHeader(http.StatusOK)
}

func processAndSendSummaries(userLinks UserLinks) {
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Println("OPENAI_API_KEY not set")
		return
	}
	summaries := make([]string, 0, len(userLinks.Links))
	for _, link := range userLinks.Links {
		summary, err := summarizeWithOpenAI(openaiKey, link)
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("%s\nError summarizing: %v", link, err))
			continue
		}
		summaries = append(summaries, fmt.Sprintf("%s\nSummary: %s", link, summary))
		// Be nice to OpenAI API
		time.Sleep(2 * time.Second)
	}
	emailBody := generateEmailBody(summaries)
	err := sendEmail(userLinks.Email, "Your Article Summaries", emailBody)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", userLinks.Email, err)
	} else {
		log.Printf("Sent summaries to %s", userLinks.Email)
	}
}

func summarizeWithOpenAI(apiKey, article string) (string, error) {
	endpoint := "https://api.openai.com/v1/responses"
	payload := map[string]interface{}{
		"model": "gpt-4.1",
		"tools": []map[string]interface{}{
			{
				"type": "web_search_preview",
			},
		},
		"input": "Please summarize this article in 2-3 sentences: " + article + " and don't include any other text or links in your response",
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("error decoding OpenAI response: %v", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", *openAIResp.Error)
	}

	if openAIResp.Status != "completed" {
		return "", fmt.Errorf("OpenAI API response not completed: %s", openAIResp.Status)
	}

	// Find the message output with the summary
	for _, output := range openAIResp.Output {
		if output.Type == "message" && output.Role == "assistant" {
			for _, content := range output.Content {
				if content.Type == "output_text" {
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no summary found in response")
}

func sendEmail(to, subject, body string) error {
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")

	// Create message
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetHeader("MIME-Version", "1.0")
	m.SetHeader("Content-Type", "text/html; charset=UTF-8")

	// Create HTML template with only inline styles
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head></head>
		<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 800px; margin: 0 auto; padding: 20px;">
			<div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 30px; text-align: center;">
				<h1 style="margin: 0; color: #333;">Your Article Summaries</h1>
				<p style="margin: 10px 0 0; color: #666;">Here are the summaries of the articles you requested:</p>
			</div>
			%s
		</body>
		</html>
	`, body)

	m.SetBody("text/html", htmlBody)

	// Create dialer
	d := gomail.NewDialer("smtp.gmail.com", 587, from, password)

	// Send email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Error sending email: %v", err)
		return err
	}

	return nil
}

func generateEmailBody(summaries []string) string {
	var body strings.Builder

	for _, summary := range summaries {
		// Split the summary into URL and content
		parts := strings.SplitN(summary, "\n", 2)
		if len(parts) != 2 {
			continue
		}
		url := parts[0]
		content := parts[1]

		// Extract the main summary and sources
		summaryParts := strings.Split(content, "\n\n##")
		mainSummary := summaryParts[0]
		var sources string
		if len(summaryParts) > 1 {
			sources = "##" + summaryParts[1]
		}

		// Convert markdown links in the summary
		mainSummary = strings.ReplaceAll(mainSummary, "[", "<a style=\"color: #0366d6; text-decoration: none;\" href=\"")
		mainSummary = strings.ReplaceAll(mainSummary, "](", "\">")
		mainSummary = strings.ReplaceAll(mainSummary, ")", "\"></a>")

		body.WriteString(fmt.Sprintf(`
			<div style="background-color: white; border: 1px solid #e9ecef; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.05);">
				<a href="%s" style="color: #0366d6; text-decoration: none; font-weight: 500; display: block; margin-bottom: 10px; word-break: break-all;">%s</a>
				<div style="color: #24292e; margin: 15px 0; font-size: 16px;">%s</div>
		`, url, url, mainSummary))

		if sources != "" {
			// Process sources section
			sources = strings.TrimPrefix(sources, "## ")
			parts := strings.SplitN(sources, ":\n", 2)
			if len(parts) == 2 {
				title := parts[0]
				links := strings.Split(parts[1], "\n- ")

				body.WriteString(fmt.Sprintf(`
					<div style="background-color: #f8f9fa; padding: 15px; border-radius: 6px; margin-top: 15px;">
						<h3 style="margin-top: 0; color: #24292e; font-size: 1.1em;">%s</h3>
						<ul style="margin: 0; padding-left: 20px;">
				`, title))

				for _, link := range links {
					link = strings.TrimSpace(link)
					if link == "" {
						continue
					}
					// Extract link text and URL
					linkParts := strings.SplitN(link, "]", 2)
					if len(linkParts) == 2 {
						linkText := strings.TrimPrefix(linkParts[0], "[")
						linkURL := strings.Trim(strings.TrimPrefix(linkParts[1], "("), ")")
						body.WriteString(fmt.Sprintf(`
							<li style="margin: 5px 0;"><a style="color: #0366d6; text-decoration: none;" href="%s">%s</a></li>
						`, linkURL, linkText))
					}
				}

				body.WriteString(`
						</ul>
					</div>
				`)
			}
		}
		body.WriteString(`
			</div>
		`)
	}

	return body.String()
}

func sanitizeFilename(email string) string {
	return strings.ReplaceAll(strings.ReplaceAll(email, "@", "_at_"), ".", "_")
}

func handleQueueCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing email", http.StatusBadRequest)
		return
	}
	userFile := filepath.Join(storageDir, sanitizeFilename(email)+".json")
	var userLinks UserLinks
	count := 0
	if data, err := os.ReadFile(userFile); err == nil {
		json.Unmarshal(data, &userLinks)
		count = len(userLinks.Links)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
