import sqlite3 as sql
import os

users_schema = """
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    partner_id INTEGER,
    public_key TEXT,
    theme TEXT DEFAULT 'light',
    wallpaper TEXT,
    receipts_enabled INTEGER DEFAULT 1,
    mfa_secret TEXT,
    mfa_enabled INTEGER DEFAULT 0,
    discord_webhook TEXT,
    FOREIGN KEY (partner_id) REFERENCES users (id)
);
"""

messages_schema = """
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sender_id INTEGER NOT NULL,
    receiver_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    message_type TEXT DEFAULT 'text',
    status TEXT DEFAULT 'sent',
    is_deleted INTEGER DEFAULT 0,
    is_edited INTEGER DEFAULT 0,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME,
    FOREIGN KEY (sender_id) REFERENCES users (id),
    FOREIGN KEY (receiver_id) REFERENCES users (id)
);
"""

def create_tables() -> None:
    try:
        data_dir = os.path.join(os.getcwd(), 'data')
        if not os.path.exists(data_dir):
            os.makedirs(data_dir)
            
        db_path = os.path.join(data_dir, 'soms.db')
        with sql.connect(db_path) as conn:
                cursor = conn.cursor()
                cursor.execute(users_schema)
                cursor.execute(messages_schema)
                conn.commit()
    except Exception as e:
        print(f"Error creating tables: {e}")
    return

def initialize_db() -> None:
    try:
        data_dir = os.path.join(os.getcwd(), 'data')
        if not os.path.exists(data_dir):
            os.makedirs(data_dir)
            
        db_path = os.path.join(data_dir, 'soms.db')
        create_tables() # Ensure base tables exist
        
        with sql.connect(db_path) as conn:
            cursor = conn.cursor()
            
            # Migration logic for users
            cursor.execute("PRAGMA table_info(users)")
            columns = [row[1] for row in cursor.fetchall()]
            if 'receipts_enabled' not in columns:
                cursor.execute("ALTER TABLE users ADD COLUMN receipts_enabled INTEGER DEFAULT 1")
            if 'mfa_secret' not in columns:
                cursor.execute("ALTER TABLE users ADD COLUMN mfa_secret TEXT")
            if 'mfa_enabled' not in columns:
                cursor.execute("ALTER TABLE users ADD COLUMN mfa_enabled INTEGER DEFAULT 0")
            if 'discord_webhook' not in columns:
                cursor.execute("ALTER TABLE users ADD COLUMN discord_webhook TEXT")
            if 'browser_notifications' not in columns:
                cursor.execute("ALTER TABLE users ADD COLUMN browser_notifications INTEGER DEFAULT 1")
            
            # Migration logic for messages
            cursor.execute("PRAGMA table_info(messages)")
            m_columns = [row[1] for row in cursor.fetchall()]
            if 'message_type' not in m_columns:
                cursor.execute("ALTER TABLE messages ADD COLUMN message_type TEXT DEFAULT 'text'")
            if 'status' not in m_columns:
                cursor.execute("ALTER TABLE messages ADD COLUMN status TEXT DEFAULT 'sent'")
            if 'read_at' not in m_columns:
                cursor.execute("ALTER TABLE messages ADD COLUMN read_at DATETIME")
            if 'is_deleted' not in m_columns:
                cursor.execute("ALTER TABLE messages ADD COLUMN is_deleted INTEGER DEFAULT 0")
            if 'is_edited' not in m_columns:
                cursor.execute("ALTER TABLE messages ADD COLUMN is_edited INTEGER DEFAULT 0")
            
            conn.commit()
    except Exception as e:
        print(f"Error initializing database: {e}")
    return

def test_db_connection():
    pass
