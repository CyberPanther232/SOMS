from flask import request, jsonify
from . import app
from .db import sql
import os
from werkzeug.security import generate_password_hash, check_password_hash
import secrets
import pyotp
import requests

def get_db_connection():
    data_dir = os.path.join(os.getcwd(), 'data')
    db_path = os.path.join(data_dir, 'soms.db')
    conn = sql.connect(db_path)
    conn.row_factory = sql.Row
    return conn

def send_discord_notification(webhook_url, sender_name):
    if not webhook_url: return
    payload = {"content": f"❤️ **SOMS**: You have a new message from **{sender_name}**!"}
    try: requests.post(webhook_url, json=payload, timeout=5)
    except Exception as e: print(f"Discord notification failed: {e}")

@app.route('/api/healthcheck')
def healthcheck(): return jsonify({"status": "OK"})

@app.route('/api/signup', methods=['POST'])
def signup():
    data = request.json
    username, email, password = data.get('username'), data.get('email'), data.get('password')
    partner_username = data.get('partner_username')
    if not username or not email or not password: return jsonify({"error": "Missing required fields"}), 400
    password_hash = generate_password_hash(password)
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("SELECT id FROM users WHERE username = ? OR email = ?", (username, email))
        if cursor.fetchone(): return jsonify({"error": "Username or email already exists"}), 409
        partner_id = None
        if partner_username:
            cursor.execute("SELECT id FROM users WHERE username = ?", (partner_username,))
            row = cursor.fetchone()
            if row: partner_id = row['id']
        cursor.execute("INSERT INTO users (username, email, password_hash, partner_id) VALUES (?, ?, ?, ?)", (username, email, password_hash, partner_id))
        user_id = cursor.lastrowid
        if partner_id: cursor.execute("UPDATE users SET partner_id = ? WHERE id = ?", (user_id, partner_id))
        conn.commit(); conn.close()
        return jsonify({"message": "User created successfully", "user_id": user_id}), 201
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/login', methods=['POST'])
def login():
    data = request.json
    username, password = data.get('username'), data.get('password')
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("SELECT * FROM users WHERE username = ?", (username,))
        user = cursor.fetchone(); conn.close()
        if user and check_password_hash(user['password_hash'], password):
            if user['mfa_enabled']: return jsonify({"mfa_required": True, "user_id": user['id']}), 200
            return jsonify({
                "message": "Login successful",
                "user": {
                    "id": user['id'], "username": user['username'], "partner_id": user['partner_id'],
                    "theme": user['theme'], "wallpaper": user['wallpaper'],
                    "receipts_enabled": bool(user['receipts_enabled']), "mfa_enabled": bool(user['mfa_enabled']),
                    "discord_webhook": user['discord_webhook'], "browser_notifications": bool(user['browser_notifications'])
                }
            }), 200
        return jsonify({"error": "Invalid credentials"}), 401
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/login/mfa', methods=['POST'])
def verify_mfa_login():
    data = request.json
    user_id, code = data.get('user_id'), data.get('code')
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("SELECT * FROM users WHERE id = ?", (user_id,))
        user = cursor.fetchone(); conn.close()
        if user and user['mfa_enabled'] and pyotp.TOTP(user['mfa_secret']).verify(code):
            return jsonify({
                "message": "Login successful",
                "user": {
                    "id": user['id'], "username": user['username'], "partner_id": user['partner_id'],
                    "theme": user['theme'], "wallpaper": user['wallpaper'],
                    "receipts_enabled": bool(user['receipts_enabled']), "mfa_enabled": True,
                    "discord_webhook": user['discord_webhook'], "browser_notifications": bool(user['browser_notifications'])
                }
            }), 200
        return jsonify({"error": "Invalid MFA code"}), 401
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/user/settings', methods=['POST'])
def update_settings():
    data = request.json
    user_id = data.get('user_id')
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        if 'theme' in data: cursor.execute("UPDATE users SET theme = ? WHERE id = ?", (data['theme'], user_id))
        if 'wallpaper' in data: cursor.execute("UPDATE users SET wallpaper = ? WHERE id = ?", (data['wallpaper'], user_id))
        if 'receipts_enabled' in data: cursor.execute("UPDATE users SET receipts_enabled = ? WHERE id = ?", (int(data['receipts_enabled']), user_id))
        if 'discord_webhook' in data: cursor.execute("UPDATE users SET discord_webhook = ? WHERE id = ?", (data['discord_webhook'], user_id))
        if 'browser_notifications' in data: cursor.execute("UPDATE users SET browser_notifications = ? WHERE id = ?", (int(data['browser_notifications']), user_id))
        conn.commit(); conn.close()
        return jsonify({"message": "Settings updated"}), 200
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/user/password', methods=['POST'])
def change_password():
    data = request.json
    u_id, old, new = data.get('user_id'), data.get('old_password'), data.get('new_password')
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("SELECT password_hash FROM users WHERE id = ?", (u_id,))
        user = cursor.fetchone()
        if user and check_password_hash(user['password_hash'], old):
            cursor.execute("UPDATE users SET password_hash = ? WHERE id = ?", (generate_password_hash(new), u_id))
            conn.commit(); conn.close(); return jsonify({"message": "OK"}), 200
        conn.close(); return jsonify({"error": "Invalid password"}), 401
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/user/mfa/setup', methods=['POST'])
def setup_mfa():
    user_id = request.json.get('user_id'); secret = pyotp.random_base32()
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("UPDATE users SET mfa_secret = ? WHERE id = ?", (secret, user_id))
        conn.commit(); conn.close(); return jsonify({"secret": secret}), 200
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/user/mfa/verify', methods=['POST'])
def verify_mfa():
    data = request.json
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("SELECT mfa_secret FROM users WHERE id = ?", (data.get('user_id'),))
        row = cursor.fetchone()
        if row and pyotp.TOTP(row['mfa_secret']).verify(data.get('code')):
            cursor.execute("UPDATE users SET mfa_enabled = 1 WHERE id = ?", (data.get('user_id'),))
            conn.commit(); conn.close(); return jsonify({"message": "OK"}), 200
        conn.close(); return jsonify({"error": "Invalid code"}), 400
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/messages', methods=['POST', 'GET'])
def messages():
    if request.method == 'POST':
        data = request.json
        s_id, r_id, content = data.get('sender_id'), data.get('receiver_id'), data.get('content')
        try:
            conn = get_db_connection(); cursor = conn.cursor()
            cursor.execute("INSERT INTO messages (sender_id, receiver_id, content, message_type) VALUES (?, ?, ?, ?)", (s_id, r_id, content, data.get('message_type', 'text')))
            cursor.execute("SELECT discord_webhook FROM users WHERE id = ?", (r_id,))
            receiver = cursor.fetchone()
            cursor.execute("SELECT username FROM users WHERE id = ?", (s_id,))
            sender = cursor.fetchone()
            if receiver and receiver['discord_webhook']: send_discord_notification(receiver['discord_webhook'], sender['username'])
            conn.commit(); conn.close(); return jsonify({"message": "OK"}), 201
        except Exception as e: return jsonify({"error": str(e)}), 500
    else:
        u_id, p_id = request.args.get('user_id'), request.args.get('partner_id')
        try:
            conn = get_db_connection(); cursor = conn.cursor()
            cursor.execute("SELECT id, receipts_enabled FROM users WHERE id IN (?, ?)", (u_id, p_id))
            r_map = {row['id']: bool(row['receipts_enabled']) for row in cursor.fetchall()}
            cursor.execute("SELECT * FROM messages WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?) ORDER BY timestamp ASC", (u_id, p_id, p_id, u_id))
            messages = [dict(row) for row in cursor.fetchall()]
            my_e, p_e = r_map.get(int(u_id), True), r_map.get(int(p_id), True)
            for m in messages:
                if not my_e or not p_e:
                    if m['status'] == 'read': m['status'] = 'sent'
            conn.close(); return jsonify(messages), 200
        except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/messages/read', methods=['POST'])
def mark_read():
    data = request.json
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("UPDATE messages SET status = 'read', read_at = CURRENT_TIMESTAMP WHERE receiver_id = ? AND sender_id = ? AND status = 'sent'", (data.get('user_id'), data.get('partner_id')))
        conn.commit(); conn.close(); return jsonify({"message": "OK"}), 200
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/messages/delete', methods=['POST'])
def delete_message():
    data = request.json
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("UPDATE messages SET is_deleted = 1, content = '' WHERE id = ? AND sender_id = ?", (data.get('message_id'), data.get('user_id')))
        conn.commit(); conn.close(); return jsonify({"message": "OK"}), 200
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/messages/edit', methods=['POST'])
def edit_message():
    data = request.json
    try:
        conn = get_db_connection(); cursor = conn.cursor()
        cursor.execute("UPDATE messages SET content = ?, is_edited = 1 WHERE id = ? AND sender_id = ?", (data.get('content'), data.get('message_id'), data.get('user_id')))
        conn.commit(); conn.close(); return jsonify({"message": "OK"}), 200
    except Exception as e: return jsonify({"error": str(e)}), 500

@app.route('/api/user/public_key', methods=['POST', 'GET'])
def manage_public_key():
    u_id = request.args.get('user_id') or request.json.get('user_id')
    if request.method == 'POST':
        try:
            conn = get_db_connection(); cursor = conn.cursor()
            cursor.execute("UPDATE users SET public_key = ? WHERE id = ?", (request.json.get('public_key'), u_id))
            conn.commit(); conn.close(); return jsonify({"message": "OK"}), 200
        except Exception as e: return jsonify({"error": str(e)}), 500
    else:
        try:
            conn = get_db_connection(); cursor = conn.cursor()
            cursor.execute("SELECT public_key FROM users WHERE id = ?", (request.args.get('target_user_id'),))
            row = cursor.fetchone(); conn.close()
            return jsonify({"public_key": row['public_key'] if row else None}), 200
        except Exception as e: return jsonify({"error": str(e)}), 500
