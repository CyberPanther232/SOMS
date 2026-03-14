package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/CyberPanther232/SOMS/client/crypto"
)

var (
	serverURL  = "http://localhost:5000"
	clientAddr = ":8080"
)

type User struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	PartnerID       int    `json:"partner_id"`
	Theme           string `json:"theme"`
	Wallpaper       string `json:"wallpaper"`
	ReceiptsEnabled bool   `json:"receipts_enabled"`
	MFAEnabled      bool   `json:"mfa_enabled"`
	DiscordWebhook  string `json:"discord_webhook"`
	PublicKey       string `json:"public_key"`
	PrivateKey      string `json:"private_key"`
}

var (
	sessions = make(map[string]*User)
	sessMu   sync.RWMutex
)

func init() {
	if url := os.Getenv("SOMS_SERVER_URL"); url != "" {
		serverURL = url
	}
	if addr := os.Getenv("SOMS_CLIENT_ADDR"); addr != "" {
		clientAddr = addr
	}
}

func main() {
	if _, err := os.Stat("keys"); os.IsNotExist(err) {
		os.Mkdir("keys", 0755)
	}

	http.HandleFunc("/api/local/signup", handleSignup)
	http.HandleFunc("/api/local/login", handleLogin)
	http.HandleFunc("/api/local/login/mfa", handleMFALogin)
	http.HandleFunc("/api/local/messages", handleMessages)
	http.HandleFunc("/api/local/messages/read", handleMarkRead)
	http.HandleFunc("/api/local/messages/delete", handleDeleteMessage)
	http.HandleFunc("/api/local/messages/edit", handleEditMessage)
	http.HandleFunc("/api/local/settings", handleSettings)
	http.HandleFunc("/api/local/status", handleStatus)
	http.HandleFunc("/api/local/logout", handleLogout)
	http.HandleFunc("/api/local/user/password", handleChangePassword)
	http.HandleFunc("/api/local/user/mfa/setup", handleMFASetup)
	http.HandleFunc("/api/local/user/mfa/verify", handleMFAVerify)

	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	fmt.Printf("SOMS Client started on http://localhost%s\n", clientAddr)
	log.Fatal(http.ListenAndServe(clientAddr, nil))
}

func getUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	sessMu.RLock()
	defer sessMu.RUnlock()
	return sessions[cookie.Value]
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(u)
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	var signupData struct {
		Username        string `json:"username"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		PartnerUsername string `json:"partner_username"`
	}
	json.NewDecoder(r.Body).Decode(&signupData)

	jsonData, _ := json.Marshal(signupData)
	resp, err := http.Post(serverURL+"/api/signup", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to connect to server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	userID := int(result["user_id"].(float64))

	pubKey, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		http.Error(w, "Failed to generate keys", http.StatusInternalServerError)
		return
	}

	savePrivateKey(signupData.Username, privKey)

	pubKeyData := map[string]interface{}{
		"user_id":    userID,
		"public_key": pubKey,
	}
	pubKeyJson, _ := json.Marshal(pubKeyData)
	http.Post(serverURL+"/api/user/public_key", "application/json", bytes.NewBuffer(pubKeyJson))

	json.NewEncoder(w).Encode(map[string]string{"message": "Signup successful! Please login."})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&loginData)

	jsonData, _ := json.Marshal(loginData)
	resp, err := http.Post(serverURL+"/api/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to connect to server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["mfa_required"] == true {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	userData := result["user"].(map[string]interface{})
	privKey, err := loadPrivateKey(loginData.Username)
	if err != nil {
		http.Error(w, "Private key not found on this device", http.StatusForbidden)
		return
	}

	u := &User{
		ID:              int(userData["id"].(float64)),
		Username:        userData["username"].(string),
		PrivateKey:      privKey,
		ReceiptsEnabled: userData["receipts_enabled"].(bool),
		MFAEnabled:      userData["mfa_enabled"].(bool),
		DiscordWebhook:  fmt.Sprintf("%v", userData["discord_webhook"]),
	}

	if theme, ok := userData["theme"].(string); ok {
		u.Theme = theme
	} else {
		u.Theme = "light"
	}
	if wallpaper, ok := userData["wallpaper"].(string); ok {
		u.Wallpaper = wallpaper
	}
	if userData["partner_id"] != nil {
		u.PartnerID = int(userData["partner_id"].(float64))
	}

	sessionID := createSession(u)
	setSessionCookie(w, sessionID)
	json.NewEncoder(w).Encode(u)
}

func handleMFALogin(w http.ResponseWriter, r *http.Request) {
	var mfaData struct {
		UserID int    `json:"user_id"`
		Code   string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&mfaData)

	jsonData, _ := json.Marshal(mfaData)
	resp, err := http.Post(serverURL+"/api/login/mfa", "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	userData := result["user"].(map[string]interface{})

	privKey, err := loadPrivateKey(userData["username"].(string))
	if err != nil {
		http.Error(w, "Private key missing", http.StatusForbidden)
		return
	}

	u := &User{
		ID:              int(userData["id"].(float64)),
		Username:        userData["username"].(string),
		PrivateKey:      privKey,
		ReceiptsEnabled: userData["receipts_enabled"].(bool),
		MFAEnabled:      userData["mfa_enabled"].(bool),
		DiscordWebhook:  fmt.Sprintf("%v", userData["discord_webhook"]),
	}
	if userData["partner_id"] != nil {
		u.PartnerID = int(userData["partner_id"].(float64))
	}

	sessionID := createSession(u)
	setSessionCookie(w, sessionID)
	json.NewEncoder(w).Encode(u)
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { http.Error(w, "Unauthorized", 401); return }
	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)
	data["user_id"] = u.ID
	jsonData, _ := json.Marshal(data)
	resp, _ := http.Post(serverURL+"/api/user/password", "application/json", bytes.NewBuffer(jsonData))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleMFASetup(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { http.Error(w, "Unauthorized", 401); return }
	jsonData, _ := json.Marshal(map[string]int{"user_id": u.ID})
	resp, _ := http.Post(serverURL+"/api/user/mfa/setup", "application/json", bytes.NewBuffer(jsonData))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleMFAVerify(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { http.Error(w, "Unauthorized", 401); return }
	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)
	data["user_id"] = u.ID
	jsonData, _ := json.Marshal(data)
	resp, _ := http.Post(serverURL+"/api/user/mfa/verify", "application/json", bytes.NewBuffer(jsonData))
	if resp.StatusCode == http.StatusOK { u.MFAEnabled = true }
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == "POST" {
		var msg struct {
			Content     string `json:"content"`
			MessageType string `json:"message_type"`
		}
		json.NewDecoder(r.Body).Decode(&msg)

		if u.PartnerID == 0 {
			http.Error(w, "No partner linked", http.StatusBadRequest)
			return
		}

		partnerPubKey, err := fetchPublicKey(u.PartnerID)
		if err != nil {
			http.Error(w, "Failed to fetch partner public key", http.StatusInternalServerError)
			return
		}

		encRecipient, err := crypto.Encrypt(msg.Content, partnerPubKey, u.PrivateKey)
		if err != nil {
			http.Error(w, "Failed to encrypt for recipient", http.StatusInternalServerError)
			return
		}

		myPubKey, _ := fetchPublicKey(u.ID)
		encSender, _ := crypto.Encrypt(msg.Content, myPubKey, u.PrivateKey)

		combinedContent := encRecipient + "|" + encSender

		sendData := map[string]interface{}{
			"sender_id":    u.ID,
			"receiver_id":  u.PartnerID,
			"content":      combinedContent,
			"message_type": msg.MessageType,
		}
		sendJson, _ := json.Marshal(sendData)
		http.Post(serverURL+"/api/messages", "application/json", bytes.NewBuffer(sendJson))
		
		w.WriteHeader(http.StatusCreated)
	} else {
		if u.PartnerID == 0 {
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		resp, err := http.Get(fmt.Sprintf("%s/api/messages?user_id=%d&partner_id=%d", serverURL, u.ID, u.PartnerID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var rawMessages []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&rawMessages)

		var decryptedMessages []map[string]interface{}
		for _, m := range rawMessages {
			if m["is_deleted"].(float64) == 1 {
				m["content"] = "[Message Deleted]"
				decryptedMessages = append(decryptedMessages, m)
				continue
			}

			content := m["content"].(string)
			senderID := int(m["sender_id"].(float64))
			
			parts := bytes.Split([]byte(content), []byte("|"))
			if len(parts) != 2 {
				m["content"] = "[Invalid Message Format]"
				decryptedMessages = append(decryptedMessages, m)
				continue
			}

			var encryptedBlob string
			var peerPubKeyID int
			if senderID == u.ID {
				encryptedBlob = string(parts[1])
				peerPubKeyID = u.ID
			} else {
				encryptedBlob = string(parts[0])
				peerPubKeyID = senderID
			}

			peerPubKey, err := fetchPublicKey(peerPubKeyID)
			if err == nil {
				decrypted, err := crypto.Decrypt(encryptedBlob, peerPubKey, u.PrivateKey)
				if err == nil {
					m["content"] = decrypted
				} else {
					m["content"] = "[Decryption Failed]"
				}
			}
			decryptedMessages = append(decryptedMessages, m)
		}
		json.NewEncoder(w).Encode(decryptedMessages)
	}
}

func handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { http.Error(w, "Unauthorized", 401); return }
	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)
	data["user_id"] = u.ID
	jsonData, _ := json.Marshal(data)
	http.Post(serverURL+"/api/messages/delete", "application/json", bytes.NewBuffer(jsonData))
	w.WriteHeader(http.StatusOK)
}

func handleEditMessage(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { http.Error(w, "Unauthorized", 401); return }
	var data struct {
		MessageID int    `json:"message_id"`
		Content   string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&data)
	partnerPubKey, _ := fetchPublicKey(u.PartnerID)
	encRecipient, _ := crypto.Encrypt(data.Content, partnerPubKey, u.PrivateKey)
	myPubKey, _ := fetchPublicKey(u.ID)
	encSender, _ := crypto.Encrypt(data.Content, myPubKey, u.PrivateKey)
	combinedContent := encRecipient + "|" + encSender
	sendData := map[string]interface{}{ "message_id": data.MessageID, "user_id": u.ID, "content": combinedContent }
	jsonData, _ := json.Marshal(sendData)
	http.Post(serverURL+"/api/messages/edit", "application/json", bytes.NewBuffer(jsonData))
	w.WriteHeader(http.StatusOK)
}

func handleMarkRead(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { return }
	jsonData, _ := json.Marshal(map[string]int{"user_id": u.ID, "partner_id": u.PartnerID})
	http.Post(serverURL+"/api/messages/read", "application/json", bytes.NewBuffer(jsonData))
	w.WriteHeader(http.StatusOK)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	u := getUser(r)
	if u == nil { return }
	var settings map[string]interface{}
	json.NewDecoder(r.Body).Decode(&settings)
	settings["user_id"] = u.ID
	jsonData, _ := json.Marshal(settings)
	http.Post(serverURL+"/api/user/settings", "application/json", bytes.NewBuffer(jsonData))
	if t, ok := settings["theme"].(string); ok { u.Theme = t }
	if w, ok := settings["wallpaper"].(string); ok { u.Wallpaper = w }
	if r, ok := settings["receipts_enabled"].(bool); ok { u.ReceiptsEnabled = r }
	if d, ok := settings["discord_webhook"].(string); ok { u.DiscordWebhook = d }
	w.WriteHeader(http.StatusOK)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		sessMu.Lock()
		delete(sessions, cookie.Value)
		sessMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusOK)
}

func createSession(u *User) string {
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	sessMu.Lock()
	sessions[id] = u
	sessMu.Unlock()
	return id
}

func setSessionCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: id, Path: "/", HttpOnly: true})
}

func fetchPublicKey(userID int) (string, error) {
	resp, _ := http.Get(fmt.Sprintf("%s/api/user/public_key?target_user_id=%d", serverURL, userID))
	defer resp.Body.Close()
	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)
	return res["public_key"], nil
}

func savePrivateKey(username string, key string) {
	os.WriteFile(filepath.Join("keys", username+".key"), []byte(key), 0600)
}

func loadPrivateKey(username string) (string, error) {
	data, err := os.ReadFile(filepath.Join("keys", username+".key"))
	return string(data), err
}
