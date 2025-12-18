package auth

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseAuth struct {
	client *auth.Client
}

func NewFirebaseAuth(ctx context.Context) (*FirebaseAuth, error) {
	// まず、GOOGLE_APPLICATION_CREDENTIALSが環境変数として設定されているか確認
	credJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	
	var app *firebase.App
	var err error
	
	if credJSON != "" && strings.HasPrefix(credJSON, "{") {
		// JSON文字列として渡された場合（Cloud Runのシークレット）
		log.Println("🔑 Initializing Firebase with credentials from environment variable (JSON string)")
		app, err = firebase.NewApp(ctx, nil, option.WithCredentialsJSON([]byte(credJSON)))
	} else if credJSON != "" {
		// ファイルパスとして渡された場合（ローカル開発）
		log.Printf("🔑 Initializing Firebase with credentials from file: %s", credJSON)
		app, err = firebase.NewApp(ctx, nil)
	} else {
		// 環境変数が設定されていない場合（デフォルト認証）
		log.Println("🔑 Initializing Firebase with default credentials")
		app, err = firebase.NewApp(ctx, nil)
	}
	
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Auth client: %w", err)
	}

	log.Println("✅ Firebase Auth initialized successfully")
	return &FirebaseAuth{client: client}, nil
}

func (f *FirebaseAuth) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	token, err := f.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("error verifying ID token: %w", err)
	}
	return token, nil
}

func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

func GetUserIDFromToken(token *auth.Token) int64 {
	uid := token.UID
	var hash int64
	for i, c := range uid {
		if i >= 16 {
			break
		}
		hash = hash*31 + int64(c)
	}
	if hash < 0 {
		hash = -hash
	}
	log.Printf("Firebase UID: %s -> user_id: %d", uid, hash)
	return hash
}

func GetPlanTypeFromToken(token *auth.Token) string {
	signInProvider := token.Firebase.SignInProvider
	log.Printf("🔍 SignInProvider: %s, UID: %s", signInProvider, token.UID)
	
	if signInProvider == "anonymous" {
		log.Printf("👤 Detected as anonymous user")
		return "free_anonymous"
	}
	
	log.Printf("👤 Detected as logged-in user (provider: %s)", signInProvider)
	return "free_login"
}

