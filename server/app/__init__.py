from flask import Flask
import os
from .db import initialize_db

app = Flask(__name__)
app.config['SECRET_KEY'] = os.getenv('SECRET_KEY', 'soms-secret-default')

initialize_db()

from app import routes
from . import api_routes
