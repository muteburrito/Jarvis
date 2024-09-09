@echo off
if not exist "chatbot" (
    echo "Creating venv..."
    python -m venv chatbot
    echo "Activating venv..."
    call chatbot\Scripts\activate
    echo "Installing deps..."
    pip install -r requirements.txt
)

if exist "__pycache__" (
    echo "Cleaning pycache before fresh run..."
    rmdir /s /q __pycache__
)

echo "Activating venv..."
call chatbot\Scripts\activate
echo "Starting Assistant..."
python assistant.py
