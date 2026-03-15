<img src="https://github.com/CyberPanther232/SOMS/blob/master/images/dark_soms_logo.png" height="500" width="750">

# SOMS - Significant Others Messaging Service

SOMS is a privacy-focused, end-to-end encrypted (E2EE) messaging application designed specifically for partners. Messages are encrypted on the client-side and can only be decrypted by the intended recipient or the sender. The server only stores encrypted blobs and has no access to message content.

## Features

- **End-to-End Encryption**: Powered by NaCl (Curve25519, XSalsa20, and Poly1305).
- **Partner Focused**: Link directly with your significant other upon signup.
- **Media Support**: Send images, GIFs (via Giphy), and videos securely.
- **Multi-Device Sync**: Robust "Double Encryption" allows both sender and receiver to access chat history from any device.
- **Read Receipts**: Know when your partner has seen your message (with privacy controls).
- **Customizable UI**: Dark/Light modes and custom chat wallpapers (upload your own!).
- **Security**: 
  - Change Password capability.
  - Multi-Factor Authentication (TOTP/2FA).
- **Notifications**:
  - Browser/Desktop notifications.
  - Discord Webhook integration.

## Architecture

- **Backend**: Python Flask (SQLite database).
- **Client**: Go (handles all cryptography and serves the Web GUI).

---

## Getting Started (Local Development)

### 1. Prerequisites
- Python 3.12+
- Go 1.25+

### 2. Setup the Server
```bash
cd server
pip install -r requirements.txt
python main.py
```
*Default: http://localhost:5000*

### 3. Setup the Client
```bash
cd client
go run main.go
```
*Default: http://localhost:8080*

---

## Containerization (Docker)

You can run the entire stack using Docker Compose:

```bash
docker-compose up --build
```

- **Client**: Accessible at `http://localhost:8080`
- **Server**: Internal communication at `http://server:5000`

---

## Configuration (Environment Variables)

### Server Options:
- `SOMS_SERVER_HOST`: Host to bind to (default: `0.0.0.0`)
- `SOMS_SERVER_PORT`: Port to listen on (default: `5000`)
- `SECRET_KEY`: Flask secret key for session security.
- `SOMS_DEBUG`: Enable debug mode (default: `True`)

### Client Options:
- `SOMS_SERVER_URL`: URL of the Python server (default: `http://localhost:5000`)
- `SOMS_CLIENT_ADDR`: Address for the local Go client to listen on (default: `:8080`)

---

## Usage Note

To test the partner linking:
1. Register **User A** and specify **User B** as the partner.
2. Register **User B** and specify **User A** as the partner.
3. Once both are registered, they will be linked and can exchange encrypted messages.

**Security Warning**: The private keys are stored locally in the `client/keys/` directory. Do not lose this folder, or you will lose access to your message history!
