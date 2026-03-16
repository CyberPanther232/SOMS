<div align="center">
  <img src="images/dark_soms_logo.png" height="300" width="600" alt="SOMS Logo">
  <h1>SOMS - Significant Others Messaging Service</h1>
  <p><i>Privacy-focused, end-to-end encrypted (E2EE) messaging for partners.</i></p>
</div>

SOMS is an encrypted messaging application designed specifically for significant others. Messages are encrypted locally on the client and only decrypted by the intended recipient or sender. The server only stores encrypted blobs, ensuring your private conversations stay private.

## ✨ Features

- 🔐 **End-to-End Encryption**: NaCl (Curve25519, XSalsa20, Poly1305) powered security.
- 👫 **Partner-Centric**: Direct linking with your partner during signup.
- 🖼️ **Media Support**: Securely share images, videos, and Giphy animations.
- 🌓 **Dynamic Themes**: Interactive Light/Dark modes with matching custom logos and favicons.
- 🎨 **Customization**: Set custom wallpapers or enjoy the new theme-specific default backgrounds.
- 🛡️ **Security Plus**: 
  - Local Private Key storage (`client/keys/`).
  - Multi-Factor Authentication (TOTP/2FA).
  - Secure Password management.
- 🔔 **Stay Notified**: Desktop notifications and Discord Webhook integration.

---

## 🚀 Quick Start (Docker)

The fastest way to get SOMS running is with **Docker Compose**. This will orchestrate both the Python backend and the Go client automatically.

### 1. Prerequisites
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

### 2. Launch SOMS
From the project root, run:
```bash
docker-compose up --build
```

### 3. Access the App
- **Web Interface**: [http://localhost:8080](http://localhost:8080)
- **API (Internal)**: [http://localhost:5000](http://localhost:5000)

---

## 🛠️ Local Development Setup

If you prefer to run the components manually:

### 1. Server (Python/Flask)
```bash
cd server
# (Optional) Create a virtual environment
python -m venv venv
source venv/bin/activate  # or venv\Scripts\activate on Windows
pip install -r requirements.txt
python main.py
```

### 2. Client (Go)
```bash
cd client
go mod download
go run main.go
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| **Server** | | |
| `SOMS_SERVER_PORT` | Port for the backend API | `5000` |
| `SECRET_KEY` | Flask session secret | `change-me` |
| `SOMS_DEBUG` | Flask debug mode | `False` |
| **Client** | | |
| `SOMS_SERVER_URL` | URL of the SOMS Server | `http://localhost:5000` |
| `SOMS_CLIENT_ADDR` | Local address to serve Web GUI | `:8080` |

---

## 🤝 How to Link Partners

1. **User A** registers and enters **User B**'s username as their partner.
2. **User B** registers and enters **User A**'s username as their partner.
3. Once both are registered, the encrypted channel is established!

> [!IMPORTANT]
> **Backup your keys!** Private keys are stored in `client/keys/`. If you delete this folder or the Docker volume associated with it, you will lose access to your decrypted message history.

---

<div align="center">
  <sub>Built with ❤️ for privacy.</sub>
</div>
