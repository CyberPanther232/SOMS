import os
from app import app

def main():
    host = os.getenv('SOMS_SERVER_HOST', '0.0.0.0')
    port = int(os.getenv('SOMS_SERVER_PORT', 5000))
    debug = os.getenv('SOMS_DEBUG', 'True').lower() == 'true'
    app.run(host=host, port=port, debug=debug)

if __name__ == '__main__':
    main()
