package send_email

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type email struct {
	Subject string `json:"subject"`
	From    string `json:"from"`
	To      string `json:"to"`
	Body    string `json:"body"`
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "./credentials/token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)

	}
	return config.Client(context.Background(), tok)
}

func SendEmail() {
	ctx := context.Background()
	b, err := os.ReadFile("./credentials/client-secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailSendScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"
	var fields email

	emailJson, err := os.Open("./email.json")
	if err != nil {
		log.Fatalf("Unable to read email file: %v", err)
	}
	rawEmailJson, _ := io.ReadAll(emailJson)
	err = json.Unmarshal(rawEmailJson, &fields)
	if err != nil {
		log.Fatalf("Unable to unmartial email file: %v", err)
	} else {
		log.Println(fields)
	}

	raw_message := []byte("From: " + fields.From + "\r\n" +
		"To: " + fields.To + "\r\n" +
		"Subject: " + fields.Subject + "\r\n\r\n" +
		fields.Body)

	var encoded_message gmail.Message

	encoded_message.Raw = base64.StdEncoding.EncodeToString(raw_message)
	encoded_message.Raw = strings.Replace(encoded_message.Raw, "/", "_", -1)
	encoded_message.Raw = strings.Replace(encoded_message.Raw, "+", "-", -1)
	encoded_message.Raw = strings.Replace(encoded_message.Raw, "=", "", -1)

	_, err = srv.Users.Messages.Send(user, &encoded_message).Do()
	if err != nil {
		log.Fatalf("Unable to send. %v", err)
	} else {
		fmt.Println("Email Sent!")
	}
}
