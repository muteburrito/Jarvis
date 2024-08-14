@echo off
if not exist "chatbot" (
    python -m venv chatbot
)
call chatbot\Scripts\activate
python app.py
